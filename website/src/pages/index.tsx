import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';

import styles from './index.module.css';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <Heading as="h1" className="hero__title">
          {siteConfig.title}
        </Heading>
        <p className="hero__subtitle">{siteConfig.tagline}</p>
        <div className={styles.buttons}>
          <Link
            className="button button--secondary button--lg"
            to="/docs/getting-started">
            Get Started
          </Link>
          <Link
            className="button button--outline button--secondary button--lg"
            style={{marginLeft: '1rem'}}
            to="https://github.com/forest6511/secretctl">
            GitHub
          </Link>
        </div>
      </div>
    </header>
  );
}

type FeatureItem = {
  title: string;
  emoji: string;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Local-First',
    emoji: '🏠',
    description: (
      <>
        Your secrets never leave your machine. No cloud accounts, no external
        services, no network requests. Just a single encrypted SQLite database.
      </>
    ),
  },
  {
    title: 'AI-Ready (MCP)',
    emoji: '🤖',
    description: (
      <>
        Built-in MCP server for secure AI agent integration. Use with Claude Code
        and other AI tools while keeping your secrets safe with Option D+.
      </>
    ),
  },
  {
    title: 'Single Binary',
    emoji: '📦',
    description: (
      <>
        Download one file and you're ready to go. No dependencies, no runtime
        requirements. Works on macOS, Linux, and Windows.
      </>
    ),
  },
];

function Feature({title, emoji, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center">
        <span style={{fontSize: '4rem'}}>{emoji}</span>
      </div>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}

function QuickInstall(): ReactNode {
  return (
    <section className={styles.quickInstall}>
      <div className="container">
        <Heading as="h2" className="text--center">
          Quick Install
        </Heading>
        <div className={styles.codeBlock}>
          <code>
            # macOS / Linux<br />
            curl -sSL https://github.com/forest6511/secretctl/releases/latest/download/secretctl-$(uname -s)-$(uname -m) -o secretctl<br />
            chmod +x secretctl<br />
            ./secretctl init
          </code>
        </div>
        <p className="text--center">
          <Link to="/docs/getting-started/installation">
            See all installation options →
          </Link>
        </p>
      </div>
    </section>
  );
}

export default function Home(): ReactNode {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title="Home"
      description="The simplest AI-ready secrets manager. Local-first, single-binary, with MCP support for AI agents.">
      <HomepageHeader />
      <main>
        <HomepageFeatures />
        <QuickInstall />
      </main>
    </Layout>
  );
}
