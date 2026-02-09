import type { ForgeConfig } from '@electron-forge/shared-types';

const config: ForgeConfig = {
  packagerConfig: {
    name: 'Mehrhof',
    executableName: 'mehrhof',
    appBundleId: 'com.valksor.mehrhof',
    asar: true,
    icon: './resources/icon',
    extraResource: ['./resources/bin'], // Bundled mehr binary (macOS/Linux only)
  },
  rebuildConfig: {},
  makers: [
    // Windows (Squirrel installer)
    {
      name: '@electron-forge/maker-squirrel',
      config: {
        name: 'Mehrhof',
        setupIcon: './resources/icon.ico',
      },
    },
    // macOS (DMG)
    {
      name: '@electron-forge/maker-dmg',
      config: {
        format: 'ULFO',
        icon: './resources/icon.icns',
      },
    },
    // Linux (DEB)
    {
      name: '@electron-forge/maker-deb',
      config: {
        options: {
          maintainer: 'Valksor',
          homepage: 'https://mehrhof.dev',
          icon: './resources/icon.png',
        },
      },
    },
    // ZIP (macOS .app bundle, Linux binary) - for direct download
    {
      name: '@electron-forge/maker-zip',
      platforms: ['darwin', 'linux'],
    },
  ],
  plugins: [],
};

export default config;
