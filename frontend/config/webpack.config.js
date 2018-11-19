const webpack = require('webpack');
const Define = webpack.DefinePlugin;
const Clean = require('clean-webpack-plugin');
const Html = require('html-webpack-plugin');

const path = x => (require('path')).resolve(__dirname, '..', ...x.split('/'));

const env = process.env.NODE_ENV || 'development';
const publicFolder = path('public');

module.exports = {
  entry: {
    remark: './src/app/app',
    counters: './src/widgets/counters/app',
    'last-comments': './src/widgets/last-comments/app',
    demo: './src/demo/demo',
  },
  output: {
    path: publicFolder,
    filename: `[name].js`,
  },
  resolve: {
    extensions: ['.jsx', '.js'],
    modules: [
      path('src'),
      path('node_modules'),
    ],
  },
  module: {
  },
  plugins: [
    new Clean(publicFolder),
    new Define({
      'process.env.NODE_ENV': JSON.stringify(env),
    }),
    new Html({
      template: path('src/demo/index.ejs'),
      inject: false,
    }),
  ],
};
