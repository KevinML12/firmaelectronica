import { browser } from '$app/environment';
import { writable } from 'svelte/store';

const STORAGE_KEY = 'oj_firma_pin_ok';
const PIN_SERVER_KEY = 'oj_pin_server';
const DAY_MS = 24 * 60 * 60 * 1000;

/** PIN esperado: variable pública de entorno (Vercel / .env). Por defecto demo. */
export function getExpectedPin(): string {
	if (!browser) return '';
	const v = import.meta.env.PUBLIC_SIGN_PIN;
	return typeof v === 'string' && v.length > 0 ? v : '2026';
}

type UnlockState = { ok: boolean; until: number };

function loadUnlock(): UnlockState {
	if (!browser) return { ok: false, until: 0 };
	try {
		const raw = sessionStorage.getItem(STORAGE_KEY);
		if (!raw) return { ok: false, until: 0 };
		const j = JSON.parse(raw) as UnlockState;
		if (j.ok && j.until > Date.now()) return j;
	} catch {
		/* ignore */
	}
	return { ok: false, until: 0 };
}

function saveUnlock(ok: boolean, pinValue?: string) {
	if (!browser) return;
	if (!ok) {
		sessionStorage.removeItem(STORAGE_KEY);
		sessionStorage.removeItem(PIN_SERVER_KEY);
		return;
	}
	const until = Date.now() + DAY_MS;
	sessionStorage.setItem(STORAGE_KEY, JSON.stringify({ ok: true, until }));
	if (pinValue != null && pinValue !== '') {
		sessionStorage.setItem(PIN_SERVER_KEY, pinValue);
	}
}

export const pinUnlocked = writable<boolean>(false);

if (browser) {
	pinUnlocked.set(loadUnlock().ok);
}

export function tryPin(pin: string): boolean {
	const expected = getExpectedPin();
	const ok = pin === expected;
	if (ok) {
		saveUnlock(true, pin);
		pinUnlocked.set(true);
	}
	return ok;
}

/** Mismo PIN para el backend (SIGN_PIN). Solo en esta sesión del navegador. */
export function getPinForServer(): string {
	if (!browser) return '';
	return sessionStorage.getItem(PIN_SERVER_KEY) ?? '';
}

export function clearPinSession() {
	saveUnlock(false);
	pinUnlocked.set(false);
}

export function isPinStillValid(): boolean {
	const s = loadUnlock();
	return s.ok && s.until > Date.now();
}

/** Refresca el store si la sesión sigue viva (al cargar rutas). */
export function refreshPinFromStorage() {
	if (browser) pinUnlocked.set(isPinStillValid());
}
