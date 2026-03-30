const base = () => (import.meta.env.PUBLIC_API_URL as string | undefined)?.replace(/\/$/, '') ?? '';

export type Juzgado = {
	codigo: string;
	nombre: string;
	departamento: string;
	municipio: string;
};

export type Hoja = {
	id: string;
	folio_numero: number;
	numero_hoja: number;
	documento_id: string;
	titulo?: string;
	tipo?: string;
};

export type DocResumen = {
	id: string;
	titulo?: string;
	page_count: number;
	storage_key: string;
	created_at: string;
};

export type ProcResumen = {
	id: string;
	documento_id: string;
	codigo_verificacion: string;
	qr_token: string;
	storage_key_salida: string;
	created_at: string;
};

export type ExpedienteDetalle = {
	id: string;
	numero_unico: string;
	tipo_proceso?: string;
	estado: string;
	cerrado_en?: string;
	checklist: Record<string, unknown>;
	juzgado: Juzgado;
	hojas: Hoja[];
	documentos: DocResumen[];
	documentos_procesados: ProcResumen[];
};

export type ExpedienteListItem = {
	id: string;
	numero_unico: string;
	tipo_proceso?: string;
	estado: string;
};

export class ApiError extends Error {
	status: number;
	body: unknown;
	constructor(message: string, status: number, body?: unknown) {
		super(message);
		this.status = status;
		this.body = body;
	}
}

async function parseJSON<T>(r: Response): Promise<T> {
	const text = await r.text();
	if (!text) return {} as T;
	try {
		return JSON.parse(text) as T;
	} catch {
		throw new ApiError('Respuesta no es JSON', r.status, text);
	}
}

export async function listExpedientes(): Promise<ExpedienteListItem[]> {
	const r = await fetch(`${base()}/api/expedientes`);
	const data = await parseJSON<ExpedienteListItem[] | { error?: string }>(r);
	if (!r.ok) throw new ApiError((data as { error?: string }).error ?? 'Error', r.status, data);
	return data as ExpedienteListItem[];
}

export async function getExpediente(id: string): Promise<ExpedienteDetalle> {
	const r = await fetch(`${base()}/api/expedientes/${encodeURIComponent(id)}`);
	const data = await parseJSON<ExpedienteDetalle & { error?: string }>(r);
	if (!r.ok) throw new ApiError(data.error ?? 'Expediente no encontrado', r.status, data);
	const d = data as ExpedienteDetalle;
	if (!d.hojas) d.hojas = [];
	if (!d.documentos) d.documentos = [];
	if (!d.documentos_procesados) d.documentos_procesados = [];
	if (!d.checklist) d.checklist = {};
	return d;
}

export async function uploadDocumento(
	expedienteId: string,
	file: File,
	titulo?: string
): Promise<{ mensaje_corto: string; folio_inicio: number; folio_fin: number; hojas: number }> {
	const fd = new FormData();
	fd.append('file', file);
	if (titulo) fd.append('titulo', titulo);
	const r = await fetch(`${base()}/api/expedientes/${encodeURIComponent(expedienteId)}/documentos`, {
		method: 'POST',
		body: fd
	});
	const data = await parseJSON<{ error?: string; mensaje_corto?: string; folio_inicio?: number; folio_fin?: number; hojas?: number }>(r);
	if (!r.ok) throw new ApiError(data.error ?? 'No se pudo subir', r.status, data);
	return data as {
		mensaje_corto: string;
		folio_inicio: number;
		folio_fin: number;
		hojas: number;
	};
}

export async function reordenarFolios(expedienteId: string, ordenIds: string[], motivo?: string): Promise<void> {
	const r = await fetch(`${base()}/api/expedientes/${encodeURIComponent(expedienteId)}/folios/reordenar`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ orden: ordenIds, motivo: motivo ?? '' })
	});
	const data = await parseJSON<{ error?: string }>(r);
	if (!r.ok) throw new ApiError(data.error ?? 'No se pudo guardar el orden', r.status, data);
}

export async function procesarDocumento(
	expedienteId: string,
	documentoId: string
): Promise<{
	documento_procesado_id: string;
	codigo_verificacion: string;
	qr_token: string;
	url_validar: string;
	url_descarga: string;
	mensaje: string;
}> {
	const r = await fetch(
		`${base()}/api/expedientes/${encodeURIComponent(expedienteId)}/documentos/${encodeURIComponent(documentoId)}/procesar`,
		{ method: 'POST' }
	);
	const data = await parseJSON<{ error?: string } & Record<string, string>>(r);
	if (!r.ok) throw new ApiError(data.error ?? 'No se pudo procesar', r.status, data);
	return data as {
		documento_procesado_id: string;
		codigo_verificacion: string;
		qr_token: string;
		url_validar: string;
		url_descarga: string;
		mensaje: string;
	};
}

export async function firmarDocumentoProcesado(
	procesadoId: string,
	body: { pin: string; rol: string; nombre_acta?: string }
): Promise<{ ok: string; nombre: string }> {
	const r = await fetch(`${base()}/api/documentos-procesados/${encodeURIComponent(procesadoId)}/firmar`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			pin: body.pin,
			rol: body.rol,
			nombre_acta: body.nombre_acta ?? ''
		})
	});
	const data = await parseJSON<{ error?: string; ok?: string; nombre?: string }>(r);
	if (!r.ok) throw new ApiError(data.error ?? 'No se pudo firmar', r.status, data);
	return data as { ok: string; nombre: string };
}

export async function cerrarExpediente(expedienteId: string, pin: string): Promise<void> {
	const r = await fetch(`${base()}/api/expedientes/${encodeURIComponent(expedienteId)}/cerrar`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ pin })
	});
	const data = await parseJSON<{ error?: string }>(r);
	if (!r.ok) throw new ApiError(data.error ?? 'No se pudo cerrar', r.status, data);
}

export function urlDescargaPDF(procesadoId: string, qrToken: string): string {
	return `${base()}/api/public/documentos-procesados/${encodeURIComponent(procesadoId)}/pdf?token=${encodeURIComponent(qrToken)}`;
}

export type ValidacionPublica = {
	valido: boolean;
	documento_procesado_id: string;
	expediente_numero: string;
	documento_titulo: string;
	codigo_verificacion: string;
	sha256: string;
	firmas: { rol: string; nombre: string; hash_interno: string; fecha: string }[];
	mensaje: string;
};

export async function fetchValidacionPublica(token: string): Promise<ValidacionPublica> {
	const r = await fetch(`${base()}/api/public/validar/${encodeURIComponent(token)}`);
	const data = await parseJSON<ValidacionPublica & { error?: string }>(r);
	if (!r.ok) throw new ApiError(data.error ?? 'No encontrado', r.status, data);
	return data as ValidacionPublica;
}

export function demoExpedienteId(): string {
	const v = import.meta.env.PUBLIC_DEMO_EXPEDIENTE_ID as string | undefined;
	return v && v.length > 0 ? v : '22222222-2222-4222-8222-222222222222';
}

export function tieneProcesado(docId: string, procesados: ProcResumen[]): boolean {
	return procesados.some((p) => p.documento_id === docId);
}
