# Code Style

Project-specific conventions that differ from defaults agents typically produce.

## Helper Functions

A well-named function improves readability even if only called once — the codebase does this (`effectivePrivate()`, `extractDomain()`). Avoid trivial wrappers that just add indirection without clarity.

## JavaScript

`const`/`let`, arrow functions, template literals, `.includes()`, spread operator. No TypeScript, no modules, no build step.

## CSS Class Naming

BEM-inspired `ComponentName_Part` pattern (e.g., `.AppCard_Icon`, `.SearchBar_Container`).
