// ESLint config — introduced as WARNINGS first (see TODO A8). The CI step runs
// with continue-on-error, so findings surface without blocking merges on the
// existing backlog. Vue 2.7 project, so the Vue 2 preset (`plugin:vue/essential`)
// is used, not the vue3-* configs.
module.exports = {
  root: true,
  env: {
    browser: true,
    node: true,
    es2021: true,
    serviceworker: true, // kill-sw.js + registered service worker
  },
  parserOptions: {
    ecmaVersion: "latest",
    sourceType: "module",
  },
  extends: ["eslint:recommended", "plugin:vue/essential"],
  rules: {
    // Keep the noisiest, backlog-heavy rules at warn so the signal stays useful
    // while we work the existing debt down.
    "no-unused-vars": "warn",
    "no-empty": "warn",
    "no-constant-condition": ["warn", { checkLoops: false }],
    "vue/no-unused-components": "warn",
    "vue/no-mutating-props": "warn",
    // View components are intentionally single-word (Home, Group, Friends…) —
    // this project doesn't follow the multi-word convention, so it's pure noise.
    "vue/multi-word-component-names": "off",
  },
};
