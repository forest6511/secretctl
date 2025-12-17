import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docs: [
    {
      type: 'category',
      label: 'Getting Started',
      collapsed: false,
      items: [
        'getting-started/index',
        'getting-started/installation',
        'getting-started/quick-start',
        'getting-started/concepts',
      ],
    },
    {
      type: 'category',
      label: 'Use Cases',
      items: [
        'use-cases/index',
        'use-cases/developer-workflows',
        'use-cases/ai-agent-integration',
      ],
    },
    {
      type: 'category',
      label: 'User Guides',
      items: [
        {
          type: 'category',
          label: 'CLI Guide',
          items: [
            'guides/cli/index',
            'guides/cli/managing-secrets',
            'guides/cli/running-commands',
            'guides/cli/exporting-secrets',
            'guides/cli/password-generation',
          ],
        },
        {
          type: 'category',
          label: 'Desktop App',
          items: [
            'guides/desktop/index',
            'guides/desktop/managing-secrets',
            'guides/desktop/keyboard-shortcuts',
            'guides/desktop/audit-logs',
          ],
        },
        {
          type: 'category',
          label: 'MCP Integration',
          items: [
            'guides/mcp/index',
            'guides/mcp/security-model',
            'guides/mcp/claude-code-setup',
            'guides/mcp/available-tools',
            'guides/mcp/env-aliases',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Comparison',
      items: [
        'comparison/index',
        'comparison/vs-1password-cli',
        'comparison/vs-vault',
        'comparison/vs-infisical',
        'comparison/feature-matrix',
      ],
    },
    {
      type: 'category',
      label: 'Migration',
      items: [
        'migration/index',
        'migration/from-env-files',
      ],
    },
    {
      type: 'category',
      label: 'Reference',
      items: [
        'reference/cli-commands',
        'reference/mcp-tools',
        'reference/configuration',
      ],
    },
    {
      type: 'category',
      label: 'Security',
      items: [
        'security/index',
        'security/how-it-works',
        'security/encryption',
      ],
    },
    {
      type: 'category',
      label: 'Contributing',
      items: [
        'contributing/index',
        'contributing/development-setup',
      ],
    },
    {
      type: 'category',
      label: 'Help',
      items: [
        'help/faq',
        'help/troubleshooting',
      ],
    },
  ],
};

export default sidebars;
