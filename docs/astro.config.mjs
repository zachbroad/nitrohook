// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	integrations: [
		starlight({
			title: 'nitrohook',
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/zachbroad/nitrohook' }],
			customCss: ['./src/styles/custom.css'],
			sidebar: [
				{
					label: 'Getting Started',
					items: [
						{ label: 'Introduction', slug: '' },
						{ label: 'Quickstart', slug: 'getting-started/quickstart' },
					],
				},
				{
					label: 'Guides',
					items: [
						{ label: 'Concepts', slug: 'guides/concepts' },
						{ label: 'Scripting', slug: 'guides/scripting' },
						{ label: 'Action Types', slug: 'guides/action-types' },
					],
				},
				{
					label: 'API Reference',
					items: [
						{ label: 'Sources', slug: 'api/sources' },
						{ label: 'Actions', slug: 'api/actions' },
						{ label: 'Deliveries', slug: 'api/deliveries' },
						{ label: 'Webhook Ingest', slug: 'api/ingest' },
					],
				},
				{
					label: 'Self-Hosting',
					items: [
						{ label: 'Configuration', slug: 'self-hosting/configuration' },
						{ label: 'Kubernetes (Helm)', slug: 'self-hosting/kubernetes' },
						{ label: 'Architecture', slug: 'self-hosting/architecture' },
					],
				},
			],
		}),
	],
});
