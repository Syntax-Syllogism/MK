import { defineConfig } from 'astro/config';
import { unified } from '@astrojs/markdown-remark';
import starlight from '@astrojs/starlight';
import starlightLinksValidator from 'starlight-links-validator';
import {
  remarkRelativeMdLinks,
  starlightPreset,
} from '@syntax-syllogism/docs-theme';

const BASE = '/mk/docs';

export default defineConfig({
  site: 'https://syntax-syllogism.com',
  base: BASE,
  outDir: './dist',
  trailingSlash: 'always',
  markdown: {
    processor: unified({
      remarkPlugins: [[remarkRelativeMdLinks, { base: BASE, docsRoot: '../docs' }]],
    }),
  },
  integrations: [
    starlight({
      ...starlightPreset({
        title: 'MK',
        base: BASE,
        toolSlug: 'mk',
        publicRepo: 'Syntax-Syllogism/MK',
        accent: 'pink',
      }),
      sidebar: [
        {
          label: 'Start here',
          items: ['index', 'getting-started'],
        },
        {
          label: 'Guides',
          items: ['web-ui', 'terminal-ui', 'columns', 'dependencies', 'git-workflow'],
        },
        {
          label: 'Reference',
          items: ['cli', 'task-files', 'http-api'],
        },
        {
          label: 'Contributing',
          items: ['development', 'board-store', 'docs-site'],
        },
      ],
    }),
    starlightLinksValidator(),
  ],
});
