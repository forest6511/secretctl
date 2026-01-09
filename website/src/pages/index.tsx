import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';
import Translate, {translate} from '@docusaurus/Translate';

import styles from './index.module.css';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <Heading as="h1" className="hero__title">
          {siteConfig.title}
        </Heading>
        <p className="hero__subtitle">
          <Translate id="homepage.tagline" description="The tagline on the homepage">
            The simplest AI-ready secrets manager
          </Translate>
        </p>
        <div className={styles.buttons}>
          <Link
            className="button button--secondary button--lg"
            to="/docs/getting-started">
            <Translate id="homepage.getStarted" description="Get Started button on homepage">
              Get Started
            </Translate>
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
    title: translate({
      id: 'homepage.features.localFirst.title',
      message: 'Local-First',
      description: 'Title for Local-First feature',
    }),
    emoji: '🏠',
    description: (
      <Translate
        id="homepage.features.localFirst.description"
        description="Description for Local-First feature">
        Your secrets never leave your machine. No cloud accounts, no external
        services, no network requests. Just a single encrypted SQLite database.
      </Translate>
    ),
  },
  {
    title: translate({
      id: 'homepage.features.aiReady.title',
      message: 'AI-Ready (MCP)',
      description: 'Title for AI-Ready feature',
    }),
    emoji: '🤖',
    description: (
      <Translate
        id="homepage.features.aiReady.description"
        description="Description for AI-Ready feature">
        Built-in MCP server for secure AI agent integration. Use with Claude Code
        and other AI tools while keeping your secrets safe with AI-Safe Access.
      </Translate>
    ),
  },
  {
    title: translate({
      id: 'homepage.features.singleBinary.title',
      message: 'Single Binary',
      description: 'Title for Single Binary feature',
    }),
    emoji: '📦',
    description: (
      <Translate
        id="homepage.features.singleBinary.description"
        description="Description for Single Binary feature">
        Download one file and you're ready to go. No dependencies, no runtime
        requirements. Works on macOS, Linux, and Windows.
      </Translate>
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
          <Translate id="homepage.quickInstall.title" description="Quick Install section title">
            Quick Install
          </Translate>
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
            <Translate id="homepage.quickInstall.seeAll" description="Link to see all installation options">
              See all installation options →
            </Translate>
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
      title={translate({
        id: 'homepage.title',
        message: 'Home',
        description: 'Homepage title',
      })}
      description={translate({
        id: 'homepage.description',
        message: 'The simplest AI-ready secrets manager. Local-first, single-binary, with MCP support for AI agents.',
        description: 'Homepage meta description',
      })}>
      <HomepageHeader />
      <main>
        <HomepageFeatures />
        <QuickInstall />
      </main>
    </Layout>
  );
}
