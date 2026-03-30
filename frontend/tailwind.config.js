/** @type {import('tailwindcss').Config} */
export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	theme: {
		extend: {
			fontFamily: {
				sans: ['system-ui', 'Segoe UI', 'Roboto', 'sans-serif']
			},
			colors: {
				oj: {
					navy: '#0c1e3d',
					gold: '#c5a059',
					paper: '#f7f4ee'
				}
			}
		}
	},
	plugins: []
};
