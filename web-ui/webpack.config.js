const path = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');
const CssMinimizerPlugin = require('css-minimizer-webpack-plugin');
const TerserPlugin = require('terser-webpack-plugin');
const CopyWebpackPlugin = require('copy-webpack-plugin');

module.exports = (env, argv) => {
  const isProduction = argv.mode === 'production';
  const outputPath = path.resolve(__dirname, '../pkg/web/static');

  return {
    entry: './src/index.tsx',
    output: {
      path: outputPath,
      filename: isProduction ? 'js/app.[contenthash].js' : 'js/[name].js',
      chunkFilename: isProduction ? 'js/[name].[contenthash].js' : 'js/[name].js',
      publicPath: '/',
      clean: {
        keep: (asset) => {
          // Keep existing static assets that are part of legacy UI
          const keepPatterns = [
            /^css\/bootstrap/,
            /^css\/fontawesome/,
            /^css\/layout\.css$/,
            /^js\/jquery/,
            /^js\/bootstrap/,
            /^js\/clipboard/,
            /^js\/color-modes/,
            /^js\/assertoor\.js$/,
            /^js\/yaml/,
            /^js\/ace/,
            /^webfonts\//,
            /^favicon\.ico$/,
            /^embed\.go$/,
          ];
          return keepPatterns.some(pattern => pattern.test(asset));
        },
      },
    },
    resolve: {
      extensions: ['.tsx', '.ts', '.js', '.jsx'],
      alias: {
        '@': path.resolve(__dirname, 'src'),
      },
    },
    module: {
      rules: [
        {
          test: /\.tsx?$/,
          use: 'ts-loader',
          exclude: /node_modules/,
        },
        {
          test: /\.css$/,
          use: [
            isProduction ? MiniCssExtractPlugin.loader : 'style-loader',
            'css-loader',
            'postcss-loader',
          ],
        },
        {
          test: /\.(png|svg|jpg|jpeg|gif|ico)$/i,
          type: 'asset/resource',
          generator: {
            filename: 'images/[name].[hash][ext]',
          },
        },
        {
          test: /\.(woff|woff2|eot|ttf|otf)$/i,
          type: 'asset/resource',
          generator: {
            filename: 'fonts/[name].[hash][ext]',
          },
        },
      ],
    },
    plugins: [
      new HtmlWebpackPlugin({
        template: './public/index.html',
        filename: 'index.html',
        inject: true,
        minify: isProduction
          ? {
              removeComments: true,
              collapseWhitespace: true,
              removeRedundantAttributes: true,
              useShortDoctype: true,
              removeEmptyAttributes: true,
              removeStyleLinkTypeAttributes: true,
              keepClosingSlash: true,
              minifyJS: true,
              minifyCSS: true,
              minifyURLs: true,
            }
          : false,
      }),
      new MiniCssExtractPlugin({
        filename: isProduction ? 'css/app.[contenthash].css' : 'css/app.css',
        chunkFilename: isProduction ? 'css/[name].[contenthash].css' : 'css/[name].css',
      }),
      new CopyWebpackPlugin({
        patterns: [
          {
            from: 'public',
            to: '',
            globOptions: {
              ignore: ['**/index.html'],
            },
          },
        ],
      }),
    ],
    optimization: {
      minimizer: [
        new TerserPlugin({
          terserOptions: {
            compress: {
              drop_console: isProduction,
            },
          },
        }),
        new CssMinimizerPlugin(),
      ],
      splitChunks: {
        chunks: 'all',
        cacheGroups: {
          vendor: {
            test: /[\\/]node_modules[\\/]/,
            name: 'vendors',
            chunks: 'all',
          },
          reactflow: {
            test: /[\\/]node_modules[\\/](reactflow|@reactflow)[\\/]/,
            name: 'reactflow',
            chunks: 'all',
            priority: 10,
          },
        },
      },
    },
    devServer: {
      static: {
        directory: outputPath,
      },
      port: 3000,
      hot: true,
      historyApiFallback: true,
      proxy: [
        {
          context: ['/api'],
          target: 'http://localhost:8080',
          changeOrigin: true,
        },
      ],
    },
    devtool: isProduction ? 'source-map' : 'eval-source-map',
  };
};
