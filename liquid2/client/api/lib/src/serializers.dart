//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_import

import 'package:one_of_serializer/any_of_serializer.dart';
import 'package:one_of_serializer/one_of_serializer.dart';
import 'package:built_collection/built_collection.dart';
import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:built_value/standard_json_plugin.dart';
import 'package:built_value/iso_8601_date_time_serializer.dart';
import 'package:liquid2_api/src/date_serializer.dart';
import 'package:liquid2_api/src/model/date.dart';

import 'package:liquid2_api/src/model/app_settings.dart';
import 'package:liquid2_api/src/model/backup_artifact.dart';
import 'package:liquid2_api/src/model/backup_output_body.dart';
import 'package:liquid2_api/src/model/blob_metadata.dart';
import 'package:liquid2_api/src/model/bookmark_document_input_body.dart';
import 'package:liquid2_api/src/model/create_export_input_body.dart';
import 'package:liquid2_api/src/model/create_feed_input_body.dart';
import 'package:liquid2_api/src/model/deleted_output_body.dart';
import 'package:liquid2_api/src/model/document_content.dart';
import 'package:liquid2_api/src/model/document_detail.dart';
import 'package:liquid2_api/src/model/document_list.dart';
import 'package:liquid2_api/src/model/document_metadata.dart';
import 'package:liquid2_api/src/model/document_note.dart';
import 'package:liquid2_api/src/model/document_summary.dart';
import 'package:liquid2_api/src/model/error_detail.dart';
import 'package:liquid2_api/src/model/error_model.dart';
import 'package:liquid2_api/src/model/export_artifact.dart';
import 'package:liquid2_api/src/model/export_output_body.dart';
import 'package:liquid2_api/src/model/feed.dart';
import 'package:liquid2_api/src/model/feed_list_output_body.dart';
import 'package:liquid2_api/src/model/feed_refresh_output_body.dart';
import 'package:liquid2_api/src/model/folder.dart';
import 'package:liquid2_api/src/model/folder_body_input_body.dart';
import 'package:liquid2_api/src/model/folder_breadcrumb.dart';
import 'package:liquid2_api/src/model/folder_list_output_body.dart';
import 'package:liquid2_api/src/model/form_file.dart';
import 'package:liquid2_api/src/model/health.dart';
import 'package:liquid2_api/src/model/job.dart';
import 'package:liquid2_api/src/model/job_list.dart';
import 'package:liquid2_api/src/model/note_body_input_body.dart';
import 'package:liquid2_api/src/model/note_list.dart';
import 'package:liquid2_api/src/model/rating_input_body.dart';
import 'package:liquid2_api/src/model/replace_tags_input_body.dart';
import 'package:liquid2_api/src/model/scrape_document_input_body.dart';
import 'package:liquid2_api/src/model/scrape_translate_document_input_body.dart';
import 'package:liquid2_api/src/model/scrape_translate_document_output_body.dart';
import 'package:liquid2_api/src/model/tag.dart';
import 'package:liquid2_api/src/model/tag_body_input_body.dart';
import 'package:liquid2_api/src/model/tag_list_output_body.dart';
import 'package:liquid2_api/src/model/translate_document_input_body.dart';
import 'package:liquid2_api/src/model/translate_document_output_body.dart';
import 'package:liquid2_api/src/model/update_document_input_body.dart';
import 'package:liquid2_api/src/model/update_feed_input_body.dart';
import 'package:liquid2_api/src/model/update_folder_input_body.dart';
import 'package:liquid2_api/src/model/update_note_input_body.dart';
import 'package:liquid2_api/src/model/update_settings_input.dart';

part 'serializers.g.dart';

@SerializersFor([
  AppSettings,
  BackupArtifact,
  BackupOutputBody,
  BlobMetadata,
  BookmarkDocumentInputBody,
  CreateExportInputBody,
  CreateFeedInputBody,
  DeletedOutputBody,
  DocumentContent,
  DocumentDetail,
  DocumentList,
  DocumentMetadata,
  DocumentNote,
  DocumentSummary,
  ErrorDetail,
  ErrorModel,
  ExportArtifact,
  ExportOutputBody,
  Feed,
  FeedListOutputBody,
  FeedRefreshOutputBody,
  Folder,
  FolderBodyInputBody,
  FolderBreadcrumb,
  FolderListOutputBody,
  FormFile,
  Health,
  Job,
  JobList,
  NoteBodyInputBody,
  NoteList,
  RatingInputBody,
  ReplaceTagsInputBody,
  ScrapeDocumentInputBody,
  ScrapeTranslateDocumentInputBody,
  ScrapeTranslateDocumentOutputBody,
  Tag,
  TagBodyInputBody,
  TagListOutputBody,
  TranslateDocumentInputBody,
  TranslateDocumentOutputBody,
  UpdateDocumentInputBody,
  UpdateFeedInputBody,
  UpdateFolderInputBody,
  UpdateNoteInputBody,
  UpdateSettingsInput,
])
Serializers serializers = (_$serializers.toBuilder()
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(String)]),
        () => ListBuilder<String>(),
      )
      ..add(const OneOfSerializer())
      ..add(const AnyOfSerializer())
      ..add(const DateSerializer())
      ..add(Iso8601DateTimeSerializer())
    ).build();

Serializers standardSerializers =
    (serializers.toBuilder()..addPlugin(StandardJsonPlugin())).build();
