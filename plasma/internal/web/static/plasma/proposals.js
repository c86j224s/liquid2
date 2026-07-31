(function proposalsNamespace(root) {
  "use strict";

  const Plasma = root.Plasma;
  const proposals = Plasma.proposals || (Plasma.proposals = {});
  const deps = {};

  function configure(values = {}) {
    Object.assign(deps, values);
  }

  function call(name, ...args) {
    const fn = deps[name];
    if (typeof fn !== "function") throw new Error(`Plasma.proposals missing dependency: ${name}`);
    return fn(...args);
  }

  Object.assign(proposals, { call, configure });
})(window);
