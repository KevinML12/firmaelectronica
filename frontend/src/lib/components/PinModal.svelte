<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { tryPin } from '$lib/pin';

	export let open = false;

	const dispatch = createEventDispatcher<{ close: void; success: void }>();

	let digits = '';
	let error = false;

	$: if (!open) {
		digits = '';
		error = false;
	}

	function append(n: string) {
		if (digits.length >= 8) return;
		digits += n;
		error = false;
	}

	function backspace() {
		digits = digits.slice(0, -1);
		error = false;
	}

	function confirmar() {
		if (tryPin(digits)) {
			digits = '';
			error = false;
			open = false;
			dispatch('success');
		} else {
			error = true;
			digits = '';
		}
	}

	function cancelar() {
		open = false;
		dispatch('close');
	}
</script>

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
		role="dialog"
		aria-modal="true"
		aria-labelledby="pin-titulo"
	>
		<div class="w-full max-w-md rounded-3xl bg-white p-6 shadow-2xl">
			<h2 id="pin-titulo" class="mb-2 text-center text-2xl font-bold text-oj-navy">
				PIN para firmar
			</h2>
			<p class="mb-6 text-center text-lg text-slate-600">
				Pidan el PIN al encargado. Nadie más debe verlo.
			</p>

			<div
				class="mb-4 flex h-14 items-center justify-center rounded-xl border-4 border-slate-300 bg-slate-50 text-3xl tracking-[0.4em] text-oj-navy"
				aria-live="polite"
			>
				{digits.replace(/./g, '●')}
			</div>

			{#if error}
				<p class="mb-4 text-center text-lg font-semibold text-red-600">PIN incorrecto. Intente de nuevo.</p>
			{/if}

			<div class="mb-4 grid grid-cols-3 gap-3">
				{#each ['1', '2', '3', '4', '5', '6', '7', '8', '9'] as n}
					<button
						type="button"
						class="btn-huge bg-slate-100 text-oj-navy hover:bg-slate-200"
						on:click={() => append(n)}>{n}</button
					>
				{/each}
				<button type="button" class="btn-huge bg-slate-200 text-oj-navy hover:bg-slate-300" on:click={backspace}
					>Borrar</button
				>
				<button type="button" class="btn-huge bg-slate-100 text-oj-navy hover:bg-slate-200" on:click={() => append('0')}
					>0</button
				>
				<button type="button" class="btn-huge btn-safe" on:click={confirmar}>Listo</button>
			</div>

			<button type="button" class="w-full rounded-xl py-4 text-lg text-slate-500 underline" on:click={cancelar}
				>Cancelar</button
			>
		</div>
	</div>
{/if}
