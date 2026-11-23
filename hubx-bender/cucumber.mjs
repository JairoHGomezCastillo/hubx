// Cucumber configuration. Step definitions and support code are TypeScript, loaded through tsx.
// One feature per story: features/HUxx_<name>.feature, scenarios copied verbatim from backlog/backlog.md.
export default {
  paths: ['features/**/*.feature'],
  import: ['tsx', 'features/support/**/*.ts', 'features/steps/**/*.ts'],
  format: ['progress', 'html:reports/cucumber.html', 'json:reports/cucumber.json'],
  formatOptions: { snippetInterface: 'async-await' },
  strict: true,
};
