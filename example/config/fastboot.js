/* global fetch */
module.exports = function (environment) {
  return {
    buildSandboxGlobals(defaultGlobals) {
      return Object.assign({}, defaultGlobals, {
        fetch: typeof fetch !== 'undefined' ? fetch : require('node-fetch'),
      });
    },
  };
};
