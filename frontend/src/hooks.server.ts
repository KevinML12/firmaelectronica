import type { Handle } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

function backendBase(): string | undefined {
	const b = env.BACKEND_URL?.trim().replace(/\/$/, '');
	return b || undefined;
}

export const handle: Handle = async ({ event, resolve }) => {
	const path = event.url.pathname;
	const isApi = path === '/api' || path.startsWith('/api/');
	if (!isApi) {
		return resolve(event);
	}

	const backend = backendBase();
	if (!backend) {
		return new Response(
			JSON.stringify({
				error:
					'Front sin BACKEND_URL: en Vercel añade BACKEND_URL = URL pública del API (Railway), sin barra final. Opcional: PUBLIC_API_URL en build para llamar al API sin proxy.'
			}),
			{ status: 503, headers: { 'Content-Type': 'application/json; charset=utf-8' } }
		);
	}

	const target = `${backend}${path}${event.url.search}`;
	const headers = new Headers(event.request.headers);
	for (const h of ['host', 'connection', 'content-length']) {
		headers.delete(h);
	}

	const init: RequestInit & { duplex?: 'half' } = {
		method: event.request.method,
		headers,
		redirect: 'manual'
	};

	if (event.request.method !== 'GET' && event.request.method !== 'HEAD') {
		init.body = event.request.body;
		init.duplex = 'half';
	}

	try {
		return await fetch(target, init);
	} catch {
		return new Response(JSON.stringify({ error: 'No se pudo contactar el API (proxy)' }), {
			status: 502,
			headers: { 'Content-Type': 'application/json; charset=utf-8' }
		});
	}
};
