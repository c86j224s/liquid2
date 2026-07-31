(function (Plasma) {
  "use strict";

  const dependencies = {};

  function configure(values) { Object.assign(dependencies, values || {}); }

  function dependency(name) {
    const value = dependencies[name];
    if (!value) throw new Error("Plasma.settings dependency missing: " + name);
    return value;
  }

  Plasma.settings = { configure, dependency };
})(window.Plasma);
