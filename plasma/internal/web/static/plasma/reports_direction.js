(function reportsDirection(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const $ = root.Plasma.dom.$;
	function currentReportDirectionHint() {
	  return $("reportDirectionHint").value;
	}

	function clearAcceptedReportDirectionHint() {
	  $("reportDirectionHint").value = "";
	}

  reports.direction = { current: currentReportDirectionHint, clear: clearAcceptedReportDirectionHint };
})(window);
