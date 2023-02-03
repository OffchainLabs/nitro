module.exports = {
  semi: false,
  trailingComma: 'es5',
  singleQuote: true,
  printWidth: 80,
  tabWidth: 2,
  arrowParens: 'avoid',
  bracketSpacing: true,
  overrides: [
    {
      files: '*.sol',
      options: {
        tabWidth: 4,
        printWidth: 100,
        singleQuote: false,
        bracketSpacing: false,
        compiler: '0.8.6',
      },
    },
  ],
}
