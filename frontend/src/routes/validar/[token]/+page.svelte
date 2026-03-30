<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { ApiError, fetchValidacionPublica, urlDescargaPDF, type ValidacionPublica } from '$lib/api/client';

	let cargando = true;
	let err = '';
	let data: ValidacionPublica | null = null;
	let token = '';

	$: token = $page.params.token ?? '';

	onMount(() => {
		void (async () => {
			try {
				data = await fetchValidacionPublica(token);
			} catch (e) {
				err = e instanceof ApiError ? e.message : 'Error de red';
			} finally {
				cargando = false;
			}
		})();
	});
</script>

<svelte:head>
	<title>Validar documento — OJ</title>
</svelte:head>

<div class="mx-auto max-w-2xl px-4 py-12">
	<h1 class="mb-6 text-3xl font-bold text-oj-navy">Validación pública</h1>

	{#if cargando}
		<p class="text-xl text-slate-600">Comprobando…</p>
	{:else if err}
		<div class="rounded-2xl border-4 border-red-200 bg-red-50 p-6 text-xl text-red-900">{err}</div>
	{:else if data}
		<div class="rounded-3xl border-2 border-emerald-200 bg-emerald-50 p-8 shadow">
			<p class="mb-2 text-sm font-bold uppercase tracking-wide text-emerald-800">Documento válido en el sistema</p>
			<p class="mb-1 text-lg text-slate-700">Expediente</p>
			<p class="mb-6 font-mono text-2xl font-bold text-oj-navy">{data.expediente_numero}</p>
			<p class="text-slate-600">{data.documento_titulo}</p>
			<p class="mt-4 font-mono text-sm text-slate-500">Código verificación: {data.codigo_verificacion}</p>
			<p class="mt-2 break-all font-mono text-xs text-slate-400">SHA-256: {data.sha256}</p>
		</div>

		<a
			href={urlDescargaPDF(data.documento_procesado_id, token)}
			target="_blank"
			rel="noopener noreferrer"
			class="btn-huge btn-primary mt-8 inline-flex w-full items-center justify-center no-underline"
		>
			Abrir / descargar PDF
		</a>

		{#if data.firmas?.length}
			<h2 class="mb-3 mt-10 text-xl font-bold text-oj-navy">Firmas registradas</h2>
			<ul class="space-y-3">
				{#each data.firmas as f}
					<li class="rounded-xl border border-slate-200 bg-white p-4">
						<p class="font-bold text-oj-navy">{f.nombre}</p>
						<p class="text-sm text-slate-600">{f.rol} · {f.fecha}</p>
						<p class="font-mono text-xs text-slate-500">{f.hash_interno}</p>
					</li>
				{/each}
			</ul>
		{/if}

		<p class="mt-8 text-slate-600">{data.mensaje}</p>
	{/if}

	<p class="mt-10">
		<a href="/" class="text-oj-gold underline">Volver al inicio</a>
	</p>
</div>
