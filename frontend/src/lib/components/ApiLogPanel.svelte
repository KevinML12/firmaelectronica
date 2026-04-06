<script lang="ts">
	import { browser } from '$app/environment';
	import { apiLogEntries, clearApiLog, type ApiLogEntry } from '$lib/api/log';
	import { getApiBase } from '$lib/api/client';

	let open = false;
	let entries: ApiLogEntry[] = [];

	$: entries = $apiLogEntries;

	function fmtTime(at: number): string {
		return new Date(at).toLocaleTimeString('es-GT', { hour12: false });
	}

	function statusClass(e: ApiLogEntry): string {
		if (!e.ok || e.status >= 400) return 'text-red-300';
		if (e.status >= 300) return 'text-amber-200';
		return 'text-emerald-300';
	}
</script>

{#if browser}
	<button
		type="button"
		class="fixed bottom-4 right-4 z-[60] rounded-full border border-white/20 bg-slate-900/90 px-4 py-2 text-sm font-medium text-slate-200 shadow-lg backdrop-blur hover:bg-slate-800"
		on:click={() => (open = !open)}
		aria-expanded={open}
	>
		API {entries.length ? `(${entries.length})` : ''}
	</button>

	{#if open}
		<div
			class="fixed inset-x-3 bottom-16 top-auto z-[59] max-h-[55vh] flex flex-col rounded-xl border border-white/15 bg-slate-950/95 shadow-2xl backdrop-blur-md md:left-auto md:right-4 md:w-[min(32rem,92vw)] md:max-h-[70vh]"
			role="dialog"
			aria-label="Registro de llamadas al API"
		>
			<div class="flex items-center justify-between border-b border-white/10 px-3 py-2">
				<div class="min-w-0 flex-1 pr-2">
					<p class="truncate text-xs font-semibold text-slate-300">Llamadas al backend</p>
					<p class="truncate font-mono text-[10px] text-slate-500" title={getApiBase() || '(vacío → URLs relativas)'}>
						Base: {getApiBase() || '— vacío —'}
					</p>
				</div>
				<button
					type="button"
					class="shrink-0 rounded-lg px-2 py-1 text-xs text-slate-400 hover:bg-white/10 hover:text-white"
					on:click={clearApiLog}>Limpiar</button
				>
			</div>
			<div class="min-h-0 flex-1 overflow-y-auto p-2 font-mono text-[11px] leading-snug">
				{#if entries.length === 0}
					<p class="px-2 py-6 text-center text-slate-500">Aún no hay peticiones en esta sesión.</p>
				{:else}
					<ul class="space-y-2">
						{#each entries as e (e.id)}
							<li class="rounded-lg bg-white/5 p-2">
								<div class="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
									<span class="text-slate-500">{fmtTime(e.at)}</span>
									<span class="font-bold text-sky-300">{e.method}</span>
									<span class={statusClass(e)}>{e.status || '—'}</span>
									<span class="text-slate-500">{e.ms}ms</span>
								</div>
								<p class="mt-1 break-all text-slate-400">{e.url}</p>
								{#if e.detail}
									<pre class="mt-1 max-h-24 overflow-auto whitespace-pre-wrap break-all text-slate-500">{e.detail}</pre>
								{/if}
							</li>
						{/each}
					</ul>
				{/if}
			</div>
		</div>
	{/if}
{/if}
