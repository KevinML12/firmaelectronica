<svelte:head>
	<title>Inicio — Expediente digital</title>
</svelte:head>

<div class="mx-auto flex max-w-4xl flex-1 flex-col justify-center gap-10 px-4 py-12">
	<div class="text-center">
		<h1 class="mb-3 text-3xl font-bold text-oj-navy md:text-4xl">¿Qué va a hacer hoy?</h1>
		<p class="text-xl text-slate-600">Elija <strong>un</strong> modo. No hay usuario ni contraseña larga.</p>
	</div>

	<div class="grid gap-6 md:grid-cols-2">
		<a href="/tramitador" class="card-tile group min-h-[220px] no-underline">
			<span class="text-6xl" aria-hidden="true">📋</span>
			<span class="text-2xl font-bold text-oj-navy">Modo tramitador</span>
			<span class="text-lg text-slate-600">Trabajo diario: expediente, folios, firmar con PIN</span>
			<span class="mt-2 text-lg font-semibold text-oj-gold"> Entrar → </span>
		</a>

		<a href="/presentacion" class="card-tile group min-h-[220px] no-underline">
			<span class="text-6xl" aria-hidden="true">🖥️</span>
			<span class="text-2xl font-bold text-oj-navy">Modo presentación</span>
			<span class="text-lg text-slate-600">Pantalla grande para el público del día (sin PIN)</span>
			<span class="mt-2 text-lg font-semibold text-oj-gold"> Abrir → </span>
		</a>
	</div>

	<p class="text-center text-sm text-slate-500">
		El PIN solo se usa para <strong>firmar</strong>, no para entrar al modo tramitador.
	</p>
</div>
