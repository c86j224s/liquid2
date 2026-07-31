(function reportsNamespace(root) {
  "use strict";
  const Plasma = root.Plasma || (root.Plasma = {});
	  const reports = Plasma.reports || (Plasma.reports = {});
	  const deps = {};
	  function configure(nextDeps) { Object.assign(deps, nextDeps || {}); }
	  function call(name, ...args) {
	    const fn = deps[name];
	    if (typeof fn !== "function") throw new Error(`Plasma.reports missing dependency: ${name}`);
	    return fn(...args);
	  }
	  Object.assign(reports, { configure, call });
	})(window);
