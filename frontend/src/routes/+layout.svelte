<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
</script>

<div class="min-h-dvh flex flex-col">
	{#if !$page.url.pathname.startsWith('/presentacion') && !$page.url.pathname.startsWith('/validar')}
		<header class="border-b border-slate-200 bg-white px-4 py-3 shadow-sm">
			<div class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-2">
				<a href="/" class="text-lg font-bold text-oj-navy md:text-xl">Expediente digital · OJ</a>
				<p class="text-sm text-slate-500">Solo toque los botones grandes</p>
			</div>
		</header>
	{/if}
	<main class="flex flex-1 flex-col">
		<slot />
	</main>
</div>
