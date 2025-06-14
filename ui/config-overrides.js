const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');

module.exports = function override(config, env) {
  // 优化Monaco Editor插件配置
  config.plugins.push(
    new MonacoWebpackPlugin({
      // 只包含实际需要的语言，减少包体积
      languages: ['json', 'yaml'],
      // 最小化功能集，大幅减少包大小
      features: ['!gotoSymbol']
    })
  );

  // 开发环境性能优化
  if (env === 'development') {
    // 禁用source map以加快编译
    config.devtool = false;

    // 优化resolve配置
    config.resolve = {
      ...config.resolve,
      symlinks: false,
      cacheWithContext: false
    };

    // 优化模块解析
    config.optimization = {
      ...config.optimization,
      removeAvailableModules: false,
      removeEmptyChunks: false,
      splitChunks: {
        ...config.optimization.splitChunks,
        cacheGroups: {
          ...config.optimization.splitChunks?.cacheGroups,
          monaco: {
            name: 'monaco',
            chunks: 'all',
            test: /[\\/]node_modules[\\/]monaco-editor[\\/]/,
            priority: 100,
            reuseExistingChunk: true
          }
        }
      }
    };

    // 添加文件系统缓存（webpack 5特性）
    if (config.cache) {
      config.cache = {
        type: 'filesystem',
        buildDependencies: {
          config: [__filename]
        }
      };
    }

    // 抑制某些警告
    config.ignoreWarnings = [
      /Failed to parse source map/,
      /Critical dependency: the request of a dependency is an expression/,
    ];
  }

  // 生产环境和开发环境都抑制source map警告
  config.ignoreWarnings = [
    ...(config.ignoreWarnings || []),
    /Failed to parse source map.*monaco-editor/,
    /ENOENT.*marked\.umd\.js\.map/,
  ];

  return config;
}; 