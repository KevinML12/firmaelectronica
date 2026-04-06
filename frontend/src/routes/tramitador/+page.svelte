<script lang="ts">
	import { onMount } from 'svelte';
	import PinModal from '$lib/components/PinModal.svelte';
	import {
		pinUnlocked,
		clearPinSession,
		refreshPinFromStorage,
		getPinForServer
	} from '$lib/pin';
	import {
		ApiError,
		demoExpedienteId,
		getExpediente,
		listExpedientes,
		listTiposDocumento,
		procesarDocumento,
		reordenarFolios,
		uploadDocumento,
		firmarDocumentoProcesado,
		cerrarExpediente,
		tieneProcesado,
		type ExpedienteDetalle,
		type Hoja,
		type TipoDocumentoCatalogo
	} from '$lib/api/client';

	let pinOpen = false;
	let pinFollowUp: (() => Promise<void>) | null = null;

	let mostrarHojas = false;
	let cargando = true;
	let subiendo = false;
	let guardandoOrden = false;
	let procesandoDoc: string | null = null;
	let firmandoProc: string | null = null;
	let cerrando = false;
	let errorMsg = '';
	let okMsg = '';
	let expediente: ExpedienteDetalle | null = null;
	let ordenLocal: Hoja[] = [];
	let ordenSucio = false;
	let fileInput: HTMLInputElement;
	let tiposDocumento: TipoDocumentoCatalogo[] = [];
	let tipoSubida = 'otro';

	$: tipoSeleccionadoCatalogo = tiposDocumento.find((x) => x.codigo === tipoSubida);

	/** Valor enviado al API (enum rol_firma_oj en BD). Vacío = no seleccionado. */
	let rolFirmar = '';

	const rolesFirma: { grupo: string; items: { value: string; label: string }[] }[] = [
		{
			grupo: 'Órgano judicial (OJ)',
			items: [
				{ value: 'juez', label: 'Juez' },
				{ value: 'secretario', label: 'Secretario(a)' },
				{ value: 'oficial_v', label: 'Oficial V' },
				{ value: 'notificador', label: 'Notificador OJ' },
				{ value: 'magistrado', label: 'Magistrado(a) sala apelaciones' },
				{ value: 'ministro_ejecutor', label: 'Ministro ejecutor' }
			]
		},
		{
			grupo: 'Laboral — partes y MTPS',
			items: [
				{ value: 'parte_actora', label: 'Parte actora / compareciente' },
				{ value: 'patrono_abogado', label: 'Abogado patrono' },
				{ value: 'representante_demandada', label: 'Representante demandada' },
				{ value: 'inspectora_trabajo', label: 'Inspectora de Trabajo (MTPS)' }
			]
		}
	];
	let nombreActa = '';
	let procSeleccionadoFirmar: string | null = null;

	onMount(() => {
		refreshPinFromStorage();
		void (async () => {
			try {
				tiposDocumento = await listTiposDocumento();
			} catch {
				tiposDocumento = [];
			}
		})();
		cargar();
	});

	async function cargar() {
		cargando = true;
		errorMsg = '';
		okMsg = '';
		try {
			const id = demoExpedienteId();
			expediente = await getExpediente(id);
		} catch (e) {
			try {
				const lista = await listExpedientes();
				if (lista.length === 0) throw new Error('No hay expedientes.');
				expediente = await getExpediente(lista[0].id);
			} catch (e2) {
				expediente = null;
				errorMsg =
					e2 instanceof ApiError
						? e2.message
						: 'No se pudo cargar. ¿Servidor y base de datos listos?';
			}
		} finally {
			cargando = false;
		}
		if (expediente) syncOrdenDesdeServidor();
	}

	function syncOrdenDesdeServidor() {
		if (!expediente) return;
		ordenLocal = [...expediente.hojas];
		ordenSucio = false;
	}

	function moveUp(i: number) {
		if (i <= 0) return;
		[ordenLocal[i - 1], ordenLocal[i]] = [ordenLocal[i], ordenLocal[i - 1]];
		ordenLocal = [...ordenLocal];
		ordenSucio = true;
	}

	function moveDown(i: number) {
		if (i >= ordenLocal.length - 1) return;
		[ordenLocal[i + 1], ordenLocal[i]] = [ordenLocal[i], ordenLocal[i + 1]];
		ordenLocal = [...ordenLocal];
		ordenSucio = true;
	}

	async function guardarOrden() {
		if (!expediente || !ordenSucio) return;
		guardandoOrden = true;
		errorMsg = '';
		okMsg = '';
		try {
			await reordenarFolios(
				expediente.id,
				ordenLocal.map((h) => h.id),
				'Tramitador web'
			);
			await cargar();
			okMsg = 'Orden de folios guardado.';
		} catch (e) {
			errorMsg = e instanceof ApiError ? e.message : 'No se guardó el orden.';
		} finally {
			guardandoOrden = false;
		}
	}

	function clickSubir() {
		fileInput?.click();
	}

	async function onFile(e: Event) {
		const el = e.target as HTMLInputElement;
		const f = el.files?.[0];
		el.value = '';
		if (!f || !expediente) return;
		if (!f.name.toLowerCase().endsWith('.pdf')) {
			errorMsg = 'Solo archivos PDF.';
			return;
		}
		subiendo = true;
		errorMsg = '';
		okMsg = '';
		try {
			const r = await uploadDocumento(expediente.id, f, f.name, tipoSubida);
			okMsg = r.mensaje_corto;
			await cargar();
			mostrarHojas = true;
		} catch (err) {
			errorMsg = err instanceof ApiError ? err.message : 'Falló la subida.';
		} finally {
			subiendo = false;
		}
	}

	async function procesar(docId: string) {
		if (!expediente || expediente.estado === 'cerrado') return;
		procesandoDoc = docId;
		errorMsg = '';
		okMsg = '';
		try {
			const r = await procesarDocumento(expediente.id, docId);
			okMsg = r.mensaje + ' Puede validar en: ' + r.url_validar;
			await cargar();
		} catch (e) {
			errorMsg = e instanceof ApiError ? e.message : 'Error al procesar.';
		} finally {
			procesandoDoc = null;
		}
	}

	function needPinThen(fn: () => Promise<void>) {
		if (getPinForServer()) {
			void fn();
			return;
		}
		pinFollowUp = fn;
		pinOpen = true;
	}

	function cerrarPin() {
		pinFollowUp = null;
		pinOpen = false;
	}

	async function alExitoPin() {
		pinOpen = false;
		if (pinFollowUp) {
			const fn = pinFollowUp;
			pinFollowUp = null;
			await fn();
		}
	}

	async function ejecutarFirma() {
		if (!procSeleccionadoFirmar || !rolFirmar.trim()) return;
		const pin = getPinForServer();
		if (!pin) {
			errorMsg = 'Meta el PIN primero (modal).';
			return;
		}
		firmandoProc = procSeleccionadoFirmar;
		errorMsg = '';
		okMsg = '';
		try {
			await firmarDocumentoProcesado(procSeleccionadoFirmar, {
				pin,
				rol: rolFirmar,
				nombre_acta: nombreActa.trim() || undefined
			});
			okMsg = 'Firma aplicada al PDF y guardada.';
			nombreActa = '';
			procSeleccionadoFirmar = null;
			await cargar();
		} catch (e) {
			errorMsg = e instanceof ApiError ? e.message : 'Error al firmar.';
		} finally {
			firmandoProc = null;
		}
	}

	function clickFirmar(procId: string) {
		procSeleccionadoFirmar = procId;
		if (!rolFirmar.trim()) {
			errorMsg = 'Elija el rol de quien firma en el acta.';
			return;
		}
		needPinThen(ejecutarFirma);
	}

	async function ejecutarCerrar() {
		if (!expediente) return;
		const pin = getPinForServer();
		if (!pin) {
			errorMsg = 'Meta el PIN para cerrar.';
			return;
		}
		cerrando = true;
		errorMsg = '';
		okMsg = '';
		try {
			await cerrarExpediente(expediente.id, pin);
			okMsg = 'Expediente cerrado. Ya quedó finiquitado en el sistema.';
			await cargar();
		} catch (e) {
			errorMsg = e instanceof ApiError ? e.message : 'No se cerró.';
		} finally {
			cerrando = false;
		}
	}

	function clickCerrarExpediente() {
		needPinThen(ejecutarCerrar);
	}

	const checklistLabels: Record<string, string> = {
		subio_pdf: 'PDF subido',
		pdf_procesado: 'PDF estampado (folios / QR)',
		firmado: 'Al menos una firma',
		expediente_cerrado: 'Expediente cerrado'
	};
</script>

<svelte:head>
	<title>Tramitador — Expediente digital</title>
</svelte:head>

<PinModal bind:open={pinOpen} on:close={cerrarPin} on:success={() => void alExitoPin()} />

<input
	bind:this={fileInput}
	type="file"
	accept="application/pdf,.pdf"
	class="sr-only"
	aria-hidden="true"
	on:change={onFile}
/>

<div class="mx-auto w-full max-w-3xl flex-1 px-4 py-8">
	<div class="mb-8 flex flex-wrap items-center justify-between gap-4">
		<div>
			<a href="/" class="mb-2 inline-block text-lg text-oj-gold underline">← Volver al inicio</a>
			<h1 class="text-3xl font-bold text-oj-navy">Modo tramitador</h1>
			<p class="text-lg text-slate-600">Siga los pasos hasta cerrar el expediente.</p>
		</div>
		{#if $pinUnlocked}
			<div class="rounded-2xl bg-emerald-100 px-4 py-3 text-center">
				<p class="font-bold text-emerald-900">✓ PIN listo para firmar y cerrar</p>
				<button
					type="button"
					class="mt-2 text-sm text-emerald-800 underline"
					on:click={() => clearPinSession()}>Borrar PIN de este navegador</button
				>
			</div>
		{/if}
	</div>

	{#if errorMsg}
		<div class="mb-6 rounded-2xl border-4 border-red-300 bg-red-50 p-6 text-xl font-semibold text-red-900">
			{errorMsg}
			<button type="button" class="btn-huge btn-primary mt-4 w-full" on:click={() => (errorMsg = '')}
				>Entendido</button
			>
		</div>
	{/if}

	{#if okMsg}
		<div class="mb-6 rounded-2xl border-4 border-emerald-300 bg-emerald-50 p-5 text-xl text-emerald-900">
			{okMsg}
		</div>
	{/if}

	{#if cargando}
		<p class="py-20 text-center text-2xl text-slate-500">Cargando expediente…</p>
	{:else if expediente}
		{#if expediente.estado === 'cerrado' || expediente.cerrado_en}
			<div class="mb-6 rounded-2xl border-4 border-oj-navy bg-slate-900 p-6 text-center text-xl text-white">
				<strong>EXPEDIENTE CERRADO</strong>
				{#if expediente.cerrado_en}
					<p class="mt-2 text-sm opacity-90">{expediente.cerrado_en}</p>
				{/if}
			</div>
		{/if}

		<section class="mb-6 rounded-2xl border-2 border-dashed border-oj-gold/50 bg-amber-50/80 p-4">
			<h2 class="mb-2 text-lg font-bold text-oj-navy">Checklist (para que no se queden a medias)</h2>
			<div class="flex flex-wrap gap-2">
				{#each Object.entries(checklistLabels) as [k, label]}
					<span
						class="rounded-full px-3 py-1 text-sm font-medium {expediente.checklist?.[k]
							? 'bg-emerald-600 text-white'
							: 'bg-slate-200 text-slate-600'}"
					>
						{expediente.checklist?.[k] ? '✓ ' : '○ '}{label}
					</span>
				{/each}
			</div>
		</section>

		<section class="mb-8 rounded-3xl border-2 border-slate-200 bg-white p-6 shadow">
			<h2 class="mb-2 text-xl font-bold text-oj-navy">Expediente</h2>
			<p class="text-2xl font-mono font-bold text-slate-900">{expediente.numero_unico}</p>
			<p class="mt-2 text-lg text-slate-700">{expediente.juzgado.nombre}</p>
			<p class="text-slate-600">{expediente.tipo_proceso || '—'} · Estado: {expediente.estado}</p>
		</section>

		<section
			class="mb-6 rounded-2xl border border-sky-200 bg-sky-50/90 p-4 text-base leading-snug text-sky-950"
			aria-label="Cómo funciona la subida"
		>
			<p class="font-bold text-sky-900">Libertad de contenido</p>
			<p class="mt-1">
				Podés subir cualquier <strong>PDF</strong> ya armado (desde Word con nuestras plantillas o no). El sistema le
				pondrá folios, código de verificación y <strong>QR</strong> hacia la página pública en Vercel al
				<strong>procesar</strong>.
			</p>
			<p class="mt-2 text-sm text-sky-900/90">
				<strong>Notificaciones OJ</strong> (casillero, constancia electrónica): la idea es que esas salgan
				<strong>generadas completas</strong> por el sistema; ese flujo aparte aún se conecta al generador automático.
			</p>
		</section>

		<section class="mb-4 rounded-2xl border border-oj-navy/20 bg-white p-4 shadow-sm">
			<label for="tipo-doc-subida" class="mb-2 block text-sm font-bold text-oj-navy"
				>Tipo de documento (alinea con plantilla DOCX en repo)</label
			>
			<select
				id="tipo-doc-subida"
				class="mb-2 w-full rounded-xl border-2 border-slate-300 bg-white p-4 text-lg"
				disabled={subiendo || expediente.estado === 'cerrado'}
				bind:value={tipoSubida}
			>
				{#if tiposDocumento.length === 0}
					<option value="otro">Otro PDF (catálogo no cargado; por defecto)</option>
				{:else}
					{#each tiposDocumento as t}
						<option value={t.codigo}>{t.etiqueta}</option>
					{/each}
				{/if}
			</select>
			{#if tipoSeleccionadoCatalogo?.plantilla_docx}
				<p class="text-sm text-slate-600">
					Plantilla referencia:
					<span class="font-mono text-oj-navy">{tipoSeleccionadoCatalogo.plantilla_docx}</span>
				</p>
			{/if}
		</section>

		<section class="grid gap-4 sm:grid-cols-2">
			<button
				type="button"
				class="btn-huge btn-primary w-full disabled:opacity-60"
				disabled={subiendo || expediente.estado === 'cerrado'}
				on:click={clickSubir}
			>
				{subiendo ? 'Subiendo…' : 'Subir PDF'}
			</button>
			<button
				type="button"
				class="btn-huge w-full bg-white text-oj-navy ring-2 ring-oj-navy hover:bg-slate-50"
				on:click={() => (mostrarHojas = !mostrarHojas)}
			>
				{mostrarHojas ? 'Ocultar' : 'Ver'} hojas / reordenar folios
			</button>
		</section>

		{#if mostrarHojas}
			{#if ordenLocal.length === 0}
				<p class="mt-6 rounded-2xl bg-amber-50 p-6 text-xl text-amber-900">Aún no hay hojas. Suba un PDF.</p>
			{:else}
				<ul class="mt-6 space-y-3 rounded-2xl bg-slate-100 p-4 text-lg">
					{#each ordenLocal as h, idx}
						<li
							class="flex flex-col gap-3 border-b border-slate-200 pb-4 last:border-0 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between"
						>
							<div class="min-w-0 flex-1">
								<span class="font-mono font-bold text-oj-navy">Folio {h.folio_numero}</span>
								<span class="ml-2 text-slate-700">{h.titulo || 'Documento'}</span>
								<p class="text-sm text-slate-500">Hoja {h.numero_hoja} del PDF</p>
							</div>
							<div class="flex shrink-0 gap-2">
								<button
									type="button"
									class="min-h-[3rem] min-w-[3.5rem] rounded-xl bg-white text-2xl font-bold shadow ring-1 ring-slate-300 disabled:opacity-30"
									disabled={idx === 0 || expediente.estado === 'cerrado'}
									on:click={() => moveUp(idx)}>↑</button
								>
								<button
									type="button"
									class="min-h-[3rem] min-w-[3.5rem] rounded-xl bg-white text-2xl font-bold shadow ring-1 ring-slate-300 disabled:opacity-30"
									disabled={idx === ordenLocal.length - 1 || expediente.estado === 'cerrado'}
									on:click={() => moveDown(idx)}>↓</button
								>
							</div>
						</li>
					{/each}
				</ul>
				<button
					type="button"
					class="btn-huge btn-warn mt-4 w-full disabled:opacity-50"
					disabled={!ordenSucio || guardandoOrden || expediente.estado === 'cerrado'}
					on:click={guardarOrden}
				>
					{guardandoOrden ? 'Guardando…' : 'Guardar orden de folios'}
				</button>
			{/if}
		{/if}

		<h2 class="mb-3 mt-10 text-2xl font-bold text-oj-navy">Documentos en el expediente</h2>
		<ul class="mb-10 space-y-4">
			{#each expediente.documentos as d}
				<li class="rounded-2xl border-2 border-slate-200 bg-white p-5 shadow-sm">
					<p class="text-lg font-bold text-slate-900">{d.titulo || 'Sin título'}</p>
					<p class="text-sm text-slate-500">{d.page_count} hojas · {d.id.slice(0, 8)}…</p>
					{#if d.tipo_etiqueta}
						<p class="mt-2 text-sm font-semibold text-oj-navy">{d.tipo_etiqueta}</p>
					{/if}
					{#if d.plantilla_docx}
						<p class="text-xs text-slate-600">
							Plantilla: <span class="font-mono">{d.plantilla_docx}</span>
						</p>
					{/if}
					{#if d.roles_sugeridos?.length}
						<p class="mt-1 text-xs text-slate-600">
							Roles sugeridos al firmar: {d.roles_sugeridos.join(', ')}
						</p>
					{/if}
					{#if tieneProcesado(d.id, expediente.documentos_procesados)}
						<p class="mt-2 font-semibold text-emerald-700">Ya procesado (estampado)</p>
					{:else}
						<button
							type="button"
							class="btn-huge btn-primary mt-3 w-full sm:w-auto"
							disabled={procesandoDoc === d.id || expediente.estado === 'cerrado'}
							on:click={() => procesar(d.id)}
						>
							{procesandoDoc === d.id ? 'Procesando…' : '1. Procesar PDF (folios + QR + rúbricas)'}
						</button>
					{/if}
				</li>
			{:else}
				<li class="text-lg text-slate-500">No hay documentos. Suba un PDF arriba.</li>
			{/each}
		</ul>

		<h2 class="mb-3 text-2xl font-bold text-oj-navy">Documentos procesados (para firma y público)</h2>
		<ul class="mb-10 space-y-6">
			{#each expediente.documentos_procesados as p}
				<li class="rounded-2xl border-2 border-oj-navy/20 bg-slate-50 p-6">
					<p class="font-mono text-sm text-slate-600">ID {p.id.slice(0, 8)}…</p>
					<p class="mt-1 font-bold text-oj-navy">Código verificación: {p.codigo_verificacion}</p>
					{#if p.tipo_etiqueta}
						<p class="mt-2 text-sm text-slate-700">{p.tipo_etiqueta}</p>
					{/if}
					{#if p.roles_sugeridos?.length}
						<p class="mt-1 text-xs text-slate-600">
							Roles sugeridos: {p.roles_sugeridos.join(', ')}
						</p>
					{/if}
					<div class="mt-4 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
						<a
							href="/validar/{p.qr_token}"
							target="_blank"
							rel="noopener noreferrer"
							class="btn-huge inline-flex items-center justify-center bg-white text-oj-navy ring-2 ring-oj-navy no-underline"
							>Abrir página de validación</a
						>
						<a
							href="{import.meta.env.PUBLIC_API_URL?.replace(/\/$/, '') ||
								''}/api/public/documentos-procesados/{p.id}/pdf?token={p.qr_token}"
							target="_blank"
							class="btn-huge inline-flex items-center justify-center bg-slate-200 text-oj-navy no-underline"
							>Descargar PDF</a
						>
					</div>

					<div class="mt-6 border-t border-slate-200 pt-6">
						<p class="mb-3 text-lg font-bold text-oj-navy">2. Firmar (última hoja del PDF)</p>
						<input
							type="text"
							bind:value={nombreActa}
							placeholder="Nombre en el acta (opcional; si vacío, usa el del rol demo)"
							class="mb-4 w-full rounded-xl border-2 border-slate-300 p-4 text-lg"
							disabled={expediente.estado === 'cerrado'}
						/>
						<label class="mb-2 block text-sm font-semibold text-slate-700" for="rol-firma-{p.id}"
							>Rol de quien firma (debe coincidir con la plantilla / acta)</label
						>
						<select
							id="rol-firma-{p.id}"
							class="mb-4 w-full rounded-xl border-2 border-slate-300 bg-white p-4 text-lg"
							disabled={expediente.estado === 'cerrado'}
							bind:value={rolFirmar}
						>
							<option value="">— Elija rol —</option>
							{#each rolesFirma as g}
								<optgroup label={g.grupo}>
									{#each g.items as it}
										<option value={it.value}>{it.label}</option>
									{/each}
								</optgroup>
							{/each}
						</select>
						<button
							type="button"
							class="btn-huge btn-safe w-full"
							disabled={!rolFirmar.trim() || firmandoProc === p.id || expediente.estado === 'cerrado'}
							on:click={() => clickFirmar(p.id)}
						>
							{firmandoProc === p.id ? 'Firmando…' : 'Firmar con PIN'}
						</button>
					</div>
				</li>
			{:else}
				<li class="text-lg text-slate-500">Procese un documento para generar el PDF con QR.</li>
			{/each}
		</ul>

		<section class="rounded-3xl border-4 border-red-200 bg-red-50 p-8">
			<h2 class="mb-2 text-2xl font-bold text-red-900">3. Cerrar expediente (fin del trámite)</h2>
			<p class="mb-6 text-lg text-red-800">
				Solo cuando ya procesó y firmó lo necesario. Pide el PIN al responsable.
			</p>
			<button
				type="button"
				class="btn-huge w-full bg-red-700 text-white hover:bg-red-800 disabled:opacity-50"
				disabled={cerrando || expediente.estado === 'cerrado'}
				on:click={clickCerrarExpediente}
			>
				{cerrando ? 'Cerrando…' : 'Cerrar expediente definitivamente'}
			</button>
		</section>
	{/if}
</div>
