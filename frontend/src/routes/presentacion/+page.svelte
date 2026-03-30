<script lang="ts">
	import { onMount } from 'svelte';
	import {
		ApiError,
		demoExpedienteId,
		getExpediente,
		listExpedientes,
		type ExpedienteDetalle,
		type Hoja
	} from '$lib/api/client';

	let cargando = true;
	let errorMsg = '';
	let expediente: ExpedienteDetalle | null = null;
	let hojas: Hoja[] = [];
	let i = 0;

	$: hoja = hojas[i] ?? null;

	onMount(() => {
		void (async () => {
			try {
				try {
					expediente = await getExpediente(demoExpedienteId());
				} catch {
					const lista = await listExpedientes();
					if (!lista.length) {
						throw new Error('No hay expedientes.');
					}
					expediente = await getExpediente(lista[0].id);
				}
				hojas = expediente?.hojas ?? [];
				i = 0;
			} catch (e) {
				errorMsg =
					e instanceof ApiError
						? e.message
						: e instanceof Error
							? e.message
							: 'Sin conexión al servidor.';
				expediente = null;
				hojas = [];
			} finally {
				cargando = false;
			}
		})();
	});

	function prev() {
		i = Math.max(0, i - 1);
	}
	function next() {
		i = Math.min(hojas.length - 1, i + 1);
	}
</script>

<svelte:head>
	<title>Presentación — {expediente?.numero_unico ?? '…'}</title>
</svelte:head>

<a
	href="/"
	class="absolute left-4 top-4 z-10 rounded-lg bg-white/10 px-3 py-2 text-sm text-white/80 backdrop-blur hover:bg-white/20"
	>Salir</a
>

{#if cargando}
	<div class="flex flex-1 items-center justify-center text-3xl text-slate-400">Cargando…</div>
{:else if errorMsg || !hoja || !expediente}
	<div class="flex flex-1 flex-col items-center justify-center gap-6 px-6 text-center">
		<p class="text-3xl text-amber-200">{errorMsg || 'No hay hojas para mostrar.'}</p>
		<a href="/tramitador" class="rounded-2xl bg-oj-gold px-8 py-4 text-xl font-bold text-oj-navy">Ir al tramitador</a>
	</div>
{:else}
	<div class="flex flex-1 flex-col items-center justify-center px-6 pb-24 pt-16 text-center">
		<p class="mb-2 text-2xl text-amber-200/90 md:text-3xl">Expediente</p>
		<p class="mb-8 font-mono text-4xl font-bold tracking-tight text-white md:text-6xl">
			{expediente.numero_unico}
		</p>

		<div
			class="w-full max-w-5xl rounded-3xl border border-white/20 bg-white/5 p-10 shadow-2xl backdrop-blur-md md:p-16"
		>
			<p class="mb-4 text-5xl font-bold text-amber-400 md:text-7xl">Folio {hoja.folio_numero}</p>
			<p class="text-2xl text-slate-200 md:text-3xl">{hoja.titulo || 'Documento'}</p>
			<p class="mt-6 text-xl text-slate-400">
				Hoja {hoja.numero_hoja} del documento · Vista pública
			</p>
		</div>
	</div>

	<div
		class="fixed bottom-0 left-0 right-0 flex items-stretch justify-center gap-3 border-t border-white/10 bg-slate-900/95 p-4 backdrop-blur"
	>
		<button
			type="button"
			class="min-h-[4rem] flex-1 max-w-xs rounded-2xl bg-slate-700 text-2xl font-bold hover:bg-slate-600 disabled:opacity-40"
			disabled={i === 0}
			on:click={prev}>← Anterior</button
		>
		<button
			type="button"
			class="min-h-[4rem] flex-1 max-w-xs rounded-2xl bg-oj-gold text-2xl font-bold text-oj-navy hover:brightness-110 disabled:opacity-40"
			disabled={i >= hojas.length - 1}
			on:click={next}>Siguiente →</button
		>
	</div>
{/if}
