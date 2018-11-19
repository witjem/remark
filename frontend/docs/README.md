# Remark42 UI

## Structure

Project structure is pretty simple:

```
.
├── config
│   └── bin
├── docs
├── node_modules
├── public
└── src
    ├── app
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
  
  * `demo` contains sources for demo page.
  
  * `widget` contains sources for widgets.
  
    * `counters` contains sources for widget,
      which shows count of comments for pages.
      
    * `last-comments` contains sources for widget,
      which shows last comments for a site.
