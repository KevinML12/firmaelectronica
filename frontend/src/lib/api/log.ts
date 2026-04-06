import { writable } from 'svelte/store';

export type ApiLogEntry = {
	id: string;
	at: number;
	method: string;
	url: string;
	status: number;
	ms: number;
	ok: boolean;
	detail?: string;
};

const MAX = 120;

export const apiLogEntries = writable<ApiLogEntry[]>([]);

export function pushApiLog(entry: Omit<ApiLogEntry, 'id' | 'at'>): void {
	const full: ApiLogEntry = {
		...entry,
		id: typeof crypto !== 'undefined' && crypto.randomUUID ? crypto.randomUUID() : String(Date.now()),
		at: Date.now()
	};
	apiLogEntries.update((list) => [full, ...list].slice(0, MAX));
}

export function clearApiLog(): void {
	apiLogEntries.set([]);
}
