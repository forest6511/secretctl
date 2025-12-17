import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'secretctl',
  tagline: 'The simplest AI-ready secrets manager',
  favicon: 'img/favicon.ico',

  // GitHub Pages deployment
  url: 'https://forest6511.github.io',
  baseUrl: '/secretctl/',
  organizationName: 'forest6511',
  projectName: 'secretctl',
  trailingSlash: false,

  onBrokenLinks: 'throw',

  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  // Analytics (Plausible - privacy-friendly)
  scripts: [
    {
      src: 'https://plausible.io/js/script.js',
      defer: true,
      'data-domain': 'forest6511.github.io',
    },
  ],

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/forest6511/secretctl/edit/main/website/',
          showLastUpdateAuthor: true,
          showLastUpdateTime: true,
        },
        blog: {
          showReadingTime: true,
          editUrl: 'https://github.com/forest6511/secretctl/edit/main/website/',
        },
        theme: {
          customCss: './src/css/custom.css',
        },
        sitemap: {
          changefreq: 'weekly' as const,
          priority: 0.5,
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: 'img/social-card.png',

    metadata: [
      {name: 'keywords', content: 'secrets manager, mcp, ai, local-first, credential management'},
      {name: 'og:image', content: '/img/social-card.png'},
    ],

    colorMode: {
      defaultMode: 'dark',
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },

    navbar: {
      title: 'secretctl',
      logo: {
        alt: 'secretctl Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docs',
          position: 'left',
          label: 'Docs',
        },
        {
          to: '/docs/guides/cli/',
          label: 'Guides',
          position: 'left',
        },
        {
          to: '/docs/reference/cli-commands',
          label: 'Reference',
          position: 'left',
        },
        {
          to: '/docs/comparison/',
          label: 'Compare',
          position: 'left',
        },
        {
          href: 'https://github.com/forest6511/secretctl',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },

    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            {label: 'Getting Started', to: '/docs/getting-started/'},
            {label: 'CLI Guide', to: '/docs/guides/cli/'},
            {label: 'MCP Integration', to: '/docs/guides/mcp/'},
          ],
        },
        {
          title: 'Compare',
          items: [
            {label: 'vs 1Password CLI', to: '/docs/comparison/vs-1password-cli'},
            {label: 'vs HashiCorp Vault', to: '/docs/comparison/vs-vault'},
            {label: 'Feature Matrix', to: '/docs/comparison/feature-matrix'},
          ],
        },
        {
          title: 'Community',
          items: [
            {label: 'GitHub', href: 'https://github.com/forest6511/secretctl'},
            {label: 'Issues', href: 'https://github.com/forest6511/secretctl/issues'},
            {label: 'Discussions', href: 'https://github.com/forest6511/secretctl/discussions'},
          ],
        },
      ],
      copyright: `Copyright ${new Date().getFullYear()} secretctl contributors.`,
    },

    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'go', 'yaml', 'json'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
