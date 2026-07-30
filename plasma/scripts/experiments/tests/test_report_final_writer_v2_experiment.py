from __future__ import annotations

import copy
from contextlib import redirect_stdout
import hashlib
import importlib.util
import inspect
import io
import json
from pathlib import Path
import sys
import tempfile
import unittest


EXPERIMENTS = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(EXPERIMENTS))
SPEC = importlib.util.spec_from_file_location(
    "report_final_writer_v2_experiment",
    EXPERIMENTS / "report_final_writer_v2_experiment.py",
)
entry = importlib.util.module_from_spec(SPEC)
assert SPEC.loader
sys.modules[SPEC.name] = entry
SPEC.loader.exec_module(entry)


def complete_pair_result(pair, winner="B", hard_fail=False, regression=False):
    row = {
        "pair_id": pair.pair_id,
        "winner": winner,
        "hard_fail": hard_fail,
        "v2_structural_regression": regression,
        "stage_trace_errors": [],
    }
    row.update({key: 0 for key in entry.HARD_FAIL_COUNT_KEYS})
    row.update({key: "pass" for key in entry.HARD_FAIL_STATUS_KEYS})
    return row


def complete_results():
    return [complete_pair_result(pair) for pair in entry.pair_specs()]


def trace_from_contract(arm, style_enabled):
    pipeline = entry.PIPELINE_V1 if arm == "A" else entry.PIPELINE_V2
    return [
        {
            "stage": stage.kind,
            "label": stage.label,
            "pipeline": pipeline,
            "tools": list(stage.tools),
            "events": list(stage.required_events),
            "fork_from": stage.fork_from,
            "source_artifact": stage.source_artifact,
            "canonicalizes": stage.canonicalizes,
        }
        for stage in entry.expected_stage_contracts(arm, style_enabled)
    ]


def write_fixed_input_receipts(archive: Path):
    for pair in entry.pair_specs():
        write_fixed_input_receipts_for_pair(archive, pair)


def write_fixed_input_receipts_for_pair(archive: Path, pair):
    manifest = archive / pair.fixed_part_manifest
    receipt = archive / pair.frozen_part_manifest_sha256_receipt
    manifest.parent.mkdir(parents=True, exist_ok=True)
    markdown = "## 검증 Part\n\n제품 상류 경로에서 검토된 한국어 Part 바이트입니다. [T-1]\n"
    prep = write_replayable_prep_fixture(archive, pair, markdown)
    record = {
        "experiment_id": entry.EXPERIMENT_ID,
        "pair_id": pair.pair_id,
        "topic_id": pair.topic_id,
        "topic_title": pair.topic_title,
        "rigor": pair.rigor,
        "source": "product_reviewed_parts_from_upstream_section_fanout",
        "prep": prep,
        "parts": [
            {
                "part_index": 1,
                "title": "검증 Part",
                "markdown": markdown,
                "sha256": hashlib.sha256(markdown.encode()).hexdigest(),
                "artifact_id": prep["part_artifact_id"],
                "word_count": 8,
            }
        ],
    }
    record["prep"].pop("part_artifact_id")
    content = json_bytes(record)
    manifest.write_bytes(content)
    digest = hashlib.sha256(content).hexdigest()
    receipt.write_text(f"{digest}  parts.manifest.json\n", encoding="utf-8")


def write_replayable_prep_fixture(archive: Path, pair, markdown: str):
    prep_dir = archive / "prep-reviewed-parts" / entry.EVIDENCE_VERSION / pair.pair_id
    prep_dir.mkdir(parents=True, exist_ok=True)
    db = prep_dir / "plasma.db"
    mission_id = f"mis_test_{pair.pair_id.replace('-', '_')}"
    pending_id = f"evt_test_{pair.pair_id.replace('-', '_')}_pending"
    plan_id = f"evt_test_{pair.pair_id.replace('-', '_')}_plan"
    source_event_id = f"evt_test_{pair.pair_id.replace('-', '_')}_source"
    source_snapshot_id = f"src_test_{pair.pair_id.replace('-', '_')}_source"
    source_artifact_id = f"art_test_{pair.pair_id.replace('-', '_')}_source"
    part_artifact_id = f"art_test_{pair.pair_id.replace('-', '_')}_part"
    source_dir = archive / "source-corpora" / pair.topic_id
    source_dir.mkdir(parents=True, exist_ok=True)
    source_bytes = "제품 상류 경로 fixture source입니다. [T-1]\n".encode("utf-8")
    (source_dir / "source-01.md").write_bytes(source_bytes)
    events = []

    def event(event_id, event_type, payload):
        events.append(
            {
                "EventID": event_id,
                "MissionID": mission_id,
                "Sequence": len(events) + 1,
                "EventType": event_type,
                "Payload": payload,
            }
        )

    event(source_event_id, "source.snapshotted", {"snapshot_id": source_snapshot_id, "artifact_ids": [source_artifact_id]})
    event(pending_id, "report.draft.pending", {"kind": "markdown_report_artifact_pending"})
    event(plan_id, "report.plan.created", {"pending_event_id": pending_id})
    for event_type in (
        "report.requirements.started",
        "report.requirements.mapped",
        "report.section.created",
        "report.part_plan.created",
        "report.part_assembly.submitted",
        "report.part.created",
        "report.part_edit.started",
    ):
        event(f"evt_test_{pair.pair_id.replace('-', '_')}_{len(events)}", event_type, {"pending_event_id": pending_id, "plan_event_id": plan_id})
    event(
        f"evt_test_{pair.pair_id.replace('-', '_')}_part_edited",
        "report.part.edited",
        {"pending_event_id": pending_id, "plan_event_id": plan_id, "artifact_id": part_artifact_id},
    )
    with entry.closing(entry.sqlite3.connect(db)) as conn, conn:
        conn.execute("create table plasma_ledger_events(event_id text, mission_id text, sequence integer, event_type text, payload_json text)")
        conn.execute("create table plasma_raw_artifacts(artifact_id text, mission_id text, sha256 text, filename text, content_blob blob)")
        conn.execute(
            "create table plasma_source_snapshots(snapshot_id text, mission_id text, connector_id text, external_source_id text, connector_version text, content_hash_value text)"
        )
        for row in events:
            conn.execute(
                "insert into plasma_ledger_events values(?,?,?,?,?)",
                (row["EventID"], row["MissionID"], row["Sequence"], row["EventType"], entry.json.dumps(row["Payload"], ensure_ascii=False, sort_keys=True)),
            )
        conn.execute(
            "insert into plasma_raw_artifacts values(?,?,?,?,?)",
            (source_artifact_id, mission_id, hashlib.sha256(source_bytes).hexdigest(), "source-01.md", source_bytes),
        )
        conn.execute(
            "insert into plasma_raw_artifacts values(?,?,?,?,?)",
            (part_artifact_id, mission_id, hashlib.sha256(markdown.encode()).hexdigest(), "part-01-edited.md", markdown.encode()),
        )
        conn.execute(
            "insert into plasma_source_snapshots values(?,?,?,?,?,?)",
            (source_snapshot_id, mission_id, "experiment-archive", f"{pair.topic_id}/source-01.md", entry.EVIDENCE_VERSION, hashlib.sha256(source_bytes).hexdigest()),
        )
    ledger = prep_dir / "ledger" / "events.json"
    ledger.parent.mkdir(parents=True, exist_ok=True)
    ledger.write_bytes(json_bytes(events))
    return {
        "product_path": "section_fanout_plan_requirement_sections_part_assembly_part_author",
        "mission_id": mission_id,
        "pending_event_id": pending_id,
        "plan_event_id": plan_id,
        "db_path": str(db),
        "ledger_events_path": str(ledger),
        "ledger_events_sha256": hashlib.sha256(ledger.read_bytes()).hexdigest(),
        "source_snapshot_ids": [source_snapshot_id],
        "source_artifact_ids": [source_artifact_id],
        "source_event_ids": [source_event_id],
        "discarded_final_report": True,
        "part_artifact_id": part_artifact_id,
    }


def json_bytes(value):
    return (entry.json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n").encode("utf-8")


class FinalWriterV2ExperimentTests(unittest.TestCase):
    def test_manifest_shape_has_four_pairs_and_frozen_inputs_for_both_arms(self):
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory) / "repo"
            repo.mkdir()
            archive = Path(directory) / "archive"
            manifest = entry.build_manifest(archive, repo_root=repo)

        self.assertEqual(manifest["experiment_id"], entry.EXPERIMENT_ID)
        self.assertEqual(manifest["evidence_version"], entry.EVIDENCE_VERSION)
        self.assertEqual(manifest["status"], "w6_b_prepared_not_run")
        self.assertEqual(manifest["public_doc_dir"], entry.PUBLIC_DOC_DIR)
        self.assertEqual(set(manifest["arms"]), {"A", "B"})
        self.assertEqual(manifest["arms"]["A"]["pipeline"], "reader_style_gate_v1")
        self.assertEqual(manifest["arms"]["B"]["pipeline"], "assembly_writer_reader_style_gate_v2")
        self.assertEqual(len(manifest["pairs"]), 4)
        self.assertEqual({row["rigor"] for row in manifest["pairs"]}, {"exploratory", "strict"})
        self.assertEqual(
            {row["topic_id"] for row in manifest["pairs"]},
            {"wang-anshi-northern-song", "go-raft-implementation-roadmap"},
        )
        for row in manifest["pairs"]:
            self.assertEqual(
                row["arm_inputs"]["A"]["frozen_part_manifest"],
                row["arm_inputs"]["B"]["frozen_part_manifest"],
            )
            self.assertEqual(
                row["arm_inputs"]["A"]["frozen_part_manifest_sha256_receipt"],
                row["arm_inputs"]["B"]["frozen_part_manifest_sha256_receipt"],
            )
            self.assertNotEqual(row["planned_outputs"]["A"]["report_markdown"], row["planned_outputs"]["B"]["report_markdown"])

    def test_adversarial_path_and_arm_tampering_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            repo = root / "repo"
            repo.mkdir()
            archive = root / "archive"
            manifest = entry.build_manifest(archive, repo_root=repo)
            entry.validate_manifest(manifest, repo_root=repo)

            with self.assertRaisesRegex(ValueError, "outside the repository"):
                entry.ensure_archive_outside_repo(repo / "plasma/docs/experiments/55", repo_root=repo)

            tampered = copy.deepcopy(manifest)
            tampered["pairs"][0]["planned_outputs"]["A"]["report_markdown"] = "/tmp/report.md"
            with self.assertRaisesRegex(ValueError, "report path changed"):
                entry.validate_manifest(tampered, repo_root=repo)

            tampered = copy.deepcopy(manifest)
            tampered["pairs"][0]["arm_inputs"]["A"]["pipeline"] = entry.PIPELINE_V2
            with self.assertRaisesRegex(ValueError, "pipeline changed"):
                entry.validate_manifest(tampered, repo_root=repo)

            tampered = copy.deepcopy(manifest)
            tampered["pairs"][0]["arm_inputs"]["B"]["frozen_part_manifest"] = "fixed-inputs/other/parts.manifest.json"
            with self.assertRaisesRegex(ValueError, "fixed Part manifest path changed"):
                entry.validate_manifest(tampered, repo_root=repo)

            tampered = copy.deepcopy(manifest)
            tampered["pairs"][0]["frozen_part_manifest_sha256_receipt"] = "../parts.manifest.sha256"
            with self.assertRaisesRegex(ValueError, "receipt path changed"):
                entry.validate_manifest(tampered, repo_root=repo)

    def test_adversarial_frozen_part_digest_mismatch_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            repo = root / "repo"
            repo.mkdir()
            archive = root / "archive"
            write_fixed_input_receipts(archive)
            manifest = entry.build_manifest(archive, repo_root=repo)
            receipts = entry.load_frozen_part_receipts(archive, repo_root=repo)
            entry.validate_fixed_input_receipts(manifest, receipts, repo_root=repo)
            with redirect_stdout(io.StringIO()):
                self.assertEqual(entry.main(["--action", "check-fixed-inputs", "--archive-root", str(archive)]), 0)

            mismatched = copy.deepcopy(receipts)
            first_pair = entry.pair_specs()[0].pair_id
            mismatched[first_pair]["B"]["frozen_part_manifest_sha256"] = "0" * 64
            with self.assertRaisesRegex(ValueError, "differs between A and B"):
                entry.validate_fixed_input_receipts(manifest, mismatched, repo_root=repo)

            receipt_path = archive / entry.pair_specs()[0].frozen_part_manifest_sha256_receipt
            receipt_path.write_text(f"{'f' * 64}  parts.manifest.json\n", encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "digest mismatch"):
                entry.load_frozen_part_receipts(archive, repo_root=repo)

    def test_adversarial_hand_authored_frozen_part_manifest_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            repo = root / "repo"
            repo.mkdir()
            archive = root / "archive"
            pair = entry.pair_specs()[0]
            manifest = archive / pair.fixed_part_manifest
            receipt = archive / pair.frozen_part_manifest_sha256_receipt
            manifest.parent.mkdir(parents=True, exist_ok=True)
            content = json_bytes(
                {
                    "experiment_id": entry.EXPERIMENT_ID,
                    "pair_id": pair.pair_id,
                    "topic_id": pair.topic_id,
                    "rigor": pair.rigor,
                    "source": "hand_authored_fixture",
                    "prep": {},
                    "parts": [],
                }
            )
            manifest.write_bytes(content)
            receipt.write_text(f"{hashlib.sha256(content).hexdigest()}  parts.manifest.json\n", encoding="utf-8")
            for other in entry.pair_specs()[1:]:
                write_fixed_input_receipts_for_pair(archive, other)

            with self.assertRaisesRegex(ValueError, "product-reviewed-Part prep"):
                entry.load_frozen_part_receipts(archive, repo_root=repo)

    def test_adversarial_duplicate_rows_and_missing_evidence_rejected(self):
        good = complete_results()
        self.assertTrue(entry.acceptance_result(good)["accepted"])

        duplicate = [good[0], copy.deepcopy(good[0]), good[2], good[3]]
        with self.assertRaisesRegex(ValueError, "duplicate pair result"):
            entry.acceptance_result(duplicate)

        for key in (
            "winner",
            "hard_fail",
            "v2_structural_regression",
            "information_loss_count",
            "product_prompt_stage_parity",
            "stage_trace_errors",
        ):
            with self.subTest(key=key):
                missing = complete_results()
                missing[0].pop(key)
                with self.assertRaises(ValueError):
                    entry.acceptance_result(missing)

        malformed = complete_results()
        malformed[0]["winner"] = "C"
        with self.assertRaisesRegex(ValueError, "winner must be A, B, or tie"):
            entry.acceptance_result(malformed)

        malformed = complete_results()
        malformed[0]["hard_fail"] = "false"
        with self.assertRaisesRegex(ValueError, "explicit boolean"):
            entry.acceptance_result(malformed)

        for value in ("0", 0.5, True):
            with self.subTest(information_loss_count=value):
                malformed = complete_results()
                malformed[0]["information_loss_count"] = value
                with self.assertRaisesRegex(ValueError, "explicit non-negative integer"):
                    entry.acceptance_result(malformed)

        malformed = complete_results()
        malformed[0]["pair_identity"] = "not_applicable"
        with self.assertRaisesRegex(ValueError, "invalid status"):
            entry.acceptance_result(malformed)

    def test_adversarial_missing_manual_adjudication_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory) / "archive"
            with self.assertRaisesRegex(ValueError, "manual adjudication is missing"):
                entry.load_manual_adjudication(archive)

            control = archive / "control"
            control.mkdir(parents=True)
            (control / "manual-adjudication.json").write_text(
                entry.json.dumps(
                    {
                        "experiment_id": entry.EXPERIMENT_ID,
                        "evidence_version": entry.EVIDENCE_VERSION,
                        "adjudicated_at": "2026-07-29T00:00:00Z",
                        "pairs": [],
                    }
                ),
                encoding="utf-8",
            )
            with self.assertRaisesRegex(ValueError, "exactly the four pairs"):
                entry.load_manual_adjudication(archive)

    def test_adversarial_manual_adjudication_rejects_malformed_structural_regression(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory) / "archive"
            write_minimal_reports(archive)
            entry.write_blind_packs(archive, seed=17)
            seal = entry.load_blind_seal(archive)
            for value in (None, "false", 0):
                record = manual_adjudication_fixture(value)
                record["blind_mapping_sha256"] = seal["blind_mapping_sha256"]
                record["blind_pack_sha256"] = seal["blind_pack_sha256"]
                entry.write_json(archive / "control" / "manual-adjudication.json", record)
                with self.subTest(value=value):
                    with self.assertRaisesRegex(ValueError, "explicit boolean"):
                        entry.load_manual_adjudication(archive)

    def test_adversarial_manual_adjudication_rejects_stale_blind_seal(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory) / "archive"
            write_minimal_reports(archive)
            entry.write_blind_packs(archive, seed=17)
            seal = entry.load_blind_seal(archive)
            record = manual_adjudication_fixture(False)
            record["blind_mapping_sha256"] = "0" * 64
            record["blind_pack_sha256"] = seal["blind_pack_sha256"]
            entry.write_json(archive / "control" / "manual-adjudication.json", record)
            with self.assertRaisesRegex(ValueError, "blind mapping digest mismatch"):
                entry.load_manual_adjudication(archive)

    def test_acceptance_rule_rejects_hard_fails_and_repeated_regression(self):
        weak = complete_results()
        weak[0]["winner"] = "A"
        weak[1]["winner"] = "A"
        result = entry.acceptance_result(weak)
        self.assertFalse(result["accepted"])
        self.assertEqual(result["equal_or_better_pairs"], 2)

        hard = complete_results()
        hard[0]["citation_loss_count"] = 1
        result = entry.acceptance_result(hard)
        self.assertFalse(result["accepted"])
        self.assertEqual(result["hard_failure_pairs"], [hard[0]["pair_id"]])

        repeated = complete_results()
        repeated[0]["v2_structural_regression"] = True
        repeated[1]["v2_structural_regression"] = True
        result = entry.acceptance_result(repeated)
        self.assertFalse(result["accepted"])
        self.assertEqual(result["repeated_structural_regression_topics"], ["wang-anshi-northern-song"])

    def test_hard_fail_detection_flags_loss_added_facts_and_contract_breaks(self):
        ok = complete_pair_result(entry.pair_specs()[0])
        self.assertEqual(entry.hard_fail_reasons(ok), [])

        bad = complete_pair_result(entry.pair_specs()[0])
        bad.update(
            {
                "information_loss_count": 1,
                "citation_loss_count": 2,
                "requirement_loss_count": 3,
                "unsupported_external_fact_count": 4,
                "product_prompt_stage_parity": "fail",
                "pair_identity": "fail",
                "blinding": "fail",
                "archive_adoption": "fail",
                "stage_payload_contract": "fail",
                "stage_trace_errors": ["wrong tool"],
            }
        )
        reasons = entry.hard_fail_reasons(bad)
        self.assertIn("information_loss_count=1", reasons)
        self.assertIn("citation_loss_count=2", reasons)
        self.assertIn("requirement_loss_count=3", reasons)
        self.assertIn("unsupported_external_fact_count=4", reasons)
        self.assertIn("product_prompt_stage_parity=fail", reasons)
        self.assertIn("pair_identity=fail", reasons)
        self.assertIn("blinding=fail", reasons)
        self.assertIn("archive_adoption=fail", reasons)
        self.assertIn("stage_payload_contract=fail", reasons)
        self.assertIn("stage_trace_errors", reasons)

    def test_stage_trace_contracts_pin_v1_and_v2_tool_partitions_and_order(self):
        self.assertEqual(entry.validate_stage_trace("A", trace_from_contract("A", style_enabled=False), style_enabled=False), [])
        self.assertEqual(entry.validate_stage_trace("B", trace_from_contract("B", style_enabled=True), style_enabled=True), [])

    def test_adversarial_missing_or_wrong_tools_rejected(self):
        trace = trace_from_contract("B", style_enabled=True)
        trace[1]["tools"] = ["plasma.report.long_form.final_write.start", "plasma.report.long_form.final_write.submit"]
        errors = entry.validate_stage_trace("B", trace, style_enabled=True)
        self.assertTrue(any("tools" in error for error in errors))

        trace = trace_from_contract("B", style_enabled=True)
        trace[1]["tools"] = list(entry.tool_surface("plasma.report.long_form.reader_edit"))
        errors = entry.validate_stage_trace("B", trace, style_enabled=True)
        self.assertTrue(any("tools" in error for error in errors))

        trace = trace_from_contract("B", style_enabled=True)
        trace[0]["tools"] = ["plasma.report.long_form.final_write.start"]
        errors = entry.validate_stage_trace("B", trace, style_enabled=True)
        self.assertTrue(any("must" in error or "tools" in error for error in errors))

        trace = trace_from_contract("B", style_enabled=True)[:-1]
        errors = entry.validate_stage_trace("B", trace, style_enabled=True)
        self.assertTrue(any("stage count" in error for error in errors))

    def test_adversarial_source_and_canonicalization_mismatch_rejected(self):
        trace = trace_from_contract("B", style_enabled=True)
        trace[2]["source_artifact"] = "final_assembly"
        errors = entry.validate_stage_trace("B", trace, style_enabled=True)
        self.assertTrue(any("source_artifact" in error for error in errors))

        trace = trace_from_contract("B", style_enabled=True)
        trace[-1]["canonicalizes"] = False
        errors = entry.validate_stage_trace("B", trace, style_enabled=True)
        self.assertTrue(any("canonicalizes" in error for error in errors))

        trace = trace_from_contract("B", style_enabled=False)
        self.assertEqual(trace[-1]["source_artifact"], "reader_edit")
        self.assertEqual(entry.validate_stage_trace("B", trace, style_enabled=False), [])

    def test_blind_mapping_is_deterministic_only_when_seed_is_injected(self):
        pairs = entry.pair_specs()
        mapping = entry.build_blind_mapping(pairs, seed=7)
        self.assertEqual(mapping, entry.build_blind_mapping(pairs, seed=7))
        for labels in mapping.values():
            self.assertEqual(set(labels), {"report_1", "report_2"})
            self.assertEqual(set(labels.values()), {"A", "B"})

    def test_adversarial_reconstructible_default_blinding_removed(self):
        self.assertFalse(hasattr(entry, "DEFAULT_BLIND_SEED"))
        signature = inspect.signature(entry.build_blind_mapping)
        self.assertIsNone(signature.parameters["seed"].default)
        args = entry.parse_args(["--action", "write-blind-packs"])
        self.assertIsNone(args.blind_seed)
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory) / "repo"
            repo.mkdir()
            manifest = entry.build_manifest(Path(directory) / "archive", repo_root=repo)
        self.assertEqual(manifest["blind_assignment"]["default"], "private_local_randomness")

    def test_existing_blind_mapping_is_reused(self):
        pairs = entry.pair_specs()
        expected = {
            pair.pair_id: {"report_1": "A", "report_2": "B"}
            if index % 2 == 0
            else {"report_1": "B", "report_2": "A"}
            for index, pair in enumerate(pairs)
        }
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "control" / "blind_mapping.json"
            entry.write_json(path, expected)
            actual = entry.load_or_create_blind_mapping(path, pairs, seed=7)
            stored = json.loads(path.read_text(encoding="utf-8"))
        self.assertEqual(actual, expected)
        self.assertEqual(stored, expected)

    def test_reading_results_are_sealed_to_mapping_and_pack_digests(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory) / "archive"
            write_minimal_reports(archive)
            entry.write_blind_packs(archive, seed=19)
            seal = entry.load_blind_seal(archive)
            result = {
                "experiment_id": entry.EXPERIMENT_ID,
                "evidence_version": entry.EVIDENCE_VERSION,
                "blind_mapping_sha256": seal["blind_mapping_sha256"],
                "blind_pack_sha256": seal["blind_pack_sha256"],
                "pairs": complete_results(),
            }
            entry.write_json(archive / "control" / "reading-results.json", result)
            self.assertEqual(entry.validate_reading_results(archive)["blind_mapping_sha256"], seal["blind_mapping_sha256"])

            pack = archive / "reading-packs" / entry.EVIDENCE_VERSION / "blind" / f"{entry.pair_specs()[0].pair_id}.md"
            pack.write_text(pack.read_text(encoding="utf-8") + "\nchanged\n", encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "pack digest mismatch"):
                entry.validate_reading_results(archive)

    def test_invalid_copy_and_seal_reuse_preserve_input_hashes(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory) / "archive"
            write_fixed_input_receipts(archive)
            write_minimal_reports(archive)
            entry.write_blind_packs(archive, seed=3)
            seal = entry.load_blind_seal(archive)
            result = {
                "experiment_id": entry.EXPERIMENT_ID,
                "evidence_version": entry.EVIDENCE_VERSION,
                "blind_mapping_sha256": seal["blind_mapping_sha256"],
                "blind_pack_sha256": seal["blind_pack_sha256"],
                "pairs": complete_results(),
            }
            entry.write_json(archive / "control" / "reading-results.json", result)
            record = manual_adjudication_fixture(False)
            record["blind_mapping_sha256"] = seal["blind_mapping_sha256"]
            record["blind_pack_sha256"] = seal["blind_pack_sha256"]
            entry.write_json(archive / "control" / "manual-adjudication.json", record)
            write_invalid_copy_fixture(archive)

            first = entry.seal_fresh_blind_packs(archive, seed=5)
            second = entry.load_blind_seal(archive)
            self.assertEqual(first, second)
            entry.verify_input_receipts_unchanged(archive)
            self.assertFalse((archive / "control" / "manual-adjudication.json").exists())
            self.assertFalse((archive / "control" / "reading-results.json").exists())

    def test_reading_pack_hides_arm_identity_and_materializes_under_archive(self):
        pairs = entry.pair_specs()
        mapping = entry.build_blind_mapping(pairs, seed=7)
        pack = entry.render_pair_reading_pack(
            pairs[0],
            mapping,
            {"A": "# First report\n\nBody.", "B": "# Second report\n\nBody."},
        )
        self.assertNotIn("reader_style_gate_v1", pack)
        self.assertNotIn("assembly_writer_reader_style_gate_v2", pack)
        self.assertIn("Report 1", pack)
        self.assertIn("Report 2", pack)

        with self.assertRaisesRegex(ValueError, "leaks arm identity"):
            entry.render_pair_reading_pack(
                pairs[0],
                mapping,
                {"A": "This came from current v1.", "B": "Clean body."},
            )

        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory) / "archive"
            for pair in pairs:
                for arm in ("A", "B"):
                    path = archive / entry.planned_report_path(pair, arm)
                    path.parent.mkdir(parents=True, exist_ok=True)
                    path.write_text(f"# Report for {pair.pair_id}\n\nBody without identity terms.", encoding="utf-8")
            written = entry.write_blind_packs(archive, seed=11)

        self.assertEqual(len(written), 5)
        self.assertTrue(any(path.name == "README.md" for path in written))
        self.assertTrue(all(path.suffix == ".md" for path in written))


def write_minimal_reports(archive: Path):
    for pair in entry.pair_specs():
        for arm in ("A", "B"):
            path = archive / entry.planned_report_path(pair, arm)
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(f"# Report for {pair.pair_id}\n\nClean report body for {arm}.", encoding="utf-8")


def manual_adjudication_fixture(regression):
    return {
        "experiment_id": entry.EXPERIMENT_ID,
        "evidence_version": entry.EVIDENCE_VERSION,
        "adjudicated_at": "2026-07-30T00:00:00Z",
        "pairs": [
            {
                "pair_id": pair.pair_id,
                "direct_reading_winner": "tie",
                "unsupported_external_facts": {"A": 0, "B": 0},
                "v2_structural_regression": regression,
                "inference_boundary": "explicit boundary",
                "reading_notes": "explicit notes",
            }
            for pair in entry.pair_specs()
        ],
    }


def write_invalid_copy_fixture(archive: Path):
    invalid = archive / entry.INVALID_UNSEALED_READING_DIR
    for rel in (
        f"control/blind_mapping.{entry.EVIDENCE_VERSION}.json",
        "control/manual-adjudication.json",
        "control/reading-results.json",
    ):
        src = archive / rel
        dst = invalid / rel
        dst.parent.mkdir(parents=True, exist_ok=True)
        dst.write_bytes(src.read_bytes())
    pack_dir = archive / "reading-packs" / entry.EVIDENCE_VERSION / "blind"
    invalid_pack_dir = invalid / "reading-packs"
    invalid_pack_dir.mkdir(parents=True, exist_ok=True)
    for path in pack_dir.glob("*.md"):
        (invalid_pack_dir / path.name).write_bytes(path.read_bytes())
    for pair in entry.pair_specs():
        for arm in ("A", "B"):
            check = archive / entry.planned_check_path(pair, arm)
            check.parent.mkdir(parents=True, exist_ok=True)
            check.write_text("{}", encoding="utf-8")
            copied = invalid / "checks" / pair.pair_id / f"{arm}.machine_check.json"
            copied.parent.mkdir(parents=True, exist_ok=True)
            copied.write_bytes(check.read_bytes())


if __name__ == "__main__":
    unittest.main()
