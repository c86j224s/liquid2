(function reportsDirection(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const $ = root.Plasma.dom.$;
	function directionControl(kind) {
	  return $(kind === "article" ? "articleDirectionHint" : "reportDirectionHint");
	}

	function currentReportDirectionHint(kind = "report") {
	  return directionControl(kind).value;
	}

	function clearAcceptedReportDirectionHint(kind = "report") {
	  directionControl(kind).value = "";
	}

  reports.direction = { current: currentReportDirectionHint, clear: clearAcceptedReportDirectionHint };
})(window);
