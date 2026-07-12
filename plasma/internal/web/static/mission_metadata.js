function missionMetadataLines(value) {
  return String(value || "").split(/\r?\n/).map((item) => item.trim()).filter(Boolean);
}

function renderMissionMetadataEditor(projection, force = false) {
  if (!projection || (!force && !$('missionMetadataForm').classList.contains('hidden'))) return;
  $('missionMetadataTitle').value = projection.title || '';
  $('missionMetadataObjective').value = projection.objective || '';
  $('missionMetadataIncluded').value = (projection.scope?.included || []).join('\n');
  $('missionMetadataExcluded').value = (projection.scope?.excluded || []).join('\n');
  $('missionMetadataEdit').disabled = !projection.mission_id;
}

document.addEventListener('DOMContentLoaded', () => {
  const form = $('missionMetadataForm');
  $('missionMetadataEdit').addEventListener('click', () => {
    if (!state.detail?.projection) return;
    form.classList.remove('hidden');
    renderMissionMetadataEditor(state.detail.projection, true);
    $('missionMetadataTitle').focus();
  });
  $('missionMetadataCancel').addEventListener('click', () => {
    form.classList.add('hidden');
    $('missionMetadataError').textContent = '';
    renderMissionMetadataEditor(state.detail?.projection, true);
  });
  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    if (!state.missionId) return;
    $('missionMetadataError').textContent = '';
    try {
      await api(`/api/missions/${encodeURIComponent(state.missionId)}`, {method: 'PATCH', body: {
        title: $('missionMetadataTitle').value,
        objective: $('missionMetadataObjective').value,
        scope: {included: missionMetadataLines($('missionMetadataIncluded').value), excluded: missionMetadataLines($('missionMetadataExcluded').value)}
      }});
      form.classList.add('hidden');
      await loadMissions();
      await reloadMission();
    } catch (err) {
      $('missionMetadataError').textContent = err.userMessage || err.message;
    }
  });
});
