# Remark42 UI

## Architecture

Remark has three applications inside:

- _main app_, built with Preact;
- _counters_, a widget written with plain JS, which shows count of comments for pages;
- _last-comments_, a widget written with plain JS, which shows last comments for a site.

## Structure

Project structure is pretty simple:

```
.
├── config
│   └── bin
├── docs
├── public
└── src
    ├── app
    │   └── components
    ├── demo
    └── widgets
        ├── counters
        └── last-comments
```

* `config` contains configuration files.
  Currently it contains only webpack configs and shell-files.
  
  * `bin` contains shell-files, that uses for run npm commands,
    which build project, starts development, etc.
    
* `docs` contains documentation of frontend part.

* `node_modules` is a typical for frontend project directory,
  which contains all dependencies. _Ignored by Git._
  
* `public` contains result files, 
  which are created during building process. _Ignored by Git._
  
* `src` contains sources.

  * `app` contains sources of main application.
  
    * `components` contains Preact components for the main application.
  
  * `demo` contains sources for demo page.
  
  * `widget` contains sources for widgets.
  
    * `counters` contains sources for widget,
      which shows count of comments for pages.
      
    * `last-comments` contains sources for widget,
      which shows last comments for a site.

## Building process

We use webpack for bundling applications. 
Also it builds demo page to make it possible to test these apps.

For transpiling JS we use babel with configuration stored in `.babelrc`,
in the root of frontend directory. 

To minify code of components we use a lot of transformations, such as:

- providing `Component` from preact as a global object `Component` using webpack's ProvidePlugin;
- wrapping imports and exports with `h` object imported from preact, using `babel-plugin-jsx-pragmatic`.

## Development

Besides everything that described in the section above, we use `react-hot-loader` for hot-loading components.

To make it work with Preact we need to create aliases for `react` and `react-dom` objects using webpack configuration. 
As an aliases value we prefer to use `preact-compat`. 
It uses only for development purposes and removes during production building 
(because `react-hot-loader` removes itself when `process.env.NODE_ENV` equals `production`).  
