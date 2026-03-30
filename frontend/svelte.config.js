import adapterNode from '@sveltejs/adapter-node';
import adapterVercel from '@sveltejs/adapter-vercel';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** Vercel: vercel.json exporta SVELTE_ADAPTER=vercel (no usar solo VERCEL: a veces está en local). */
const adapter =
	process.env.SVELTE_ADAPTER === 'vercel' ? adapterVercel() : adapterNode();

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter
	}
};

export default config;
