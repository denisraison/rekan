import type { IncomingMessage, ServerResponse } from 'node:http';
import type { Plugin } from 'vite';

interface MockRoute {
	method: string;
	pattern: RegExp;
	delay: number;
	response: unknown;
}

const MOCK_POST = {
	caption:
		'Cabelo lavado, cortado e estilizado com carinho. ' +
		'Agende seu horario e venha se sentir incrivel! ' +
		'Nosso salao esta de portas abertas pra voce.',
	hashtags: ['#salao', '#beleza', '#cabeloperfeito', '#autoestima'],
	production_note:
		'Foto da cliente sorrindo no espelho apos o corte, luz natural da janela do salao.',
};

const MOCK_IDEAS = [
	{
		caption:
			'Segunda-feira e dia de renovar o visual! Venha conhecer nossos tratamentos capilares.',
		hashtags: ['#segunda', '#salao', '#tratamento'],
		production_note: 'Foto de produtos capilares organizados na bancada.',
	},
	{
		caption: 'Transformacao do dia: de cabelo sem vida pra um look poderoso. Agenda aberta!',
		hashtags: ['#antesedepois', '#transformacao', '#salao'],
		production_note: 'Colagem antes/depois da cliente, fundo neutro do salao.',
	},
	{
		caption: 'Sexta-feira merece um cabelo a altura! Ultimos horarios disponiveis.',
		hashtags: ['#sextou', '#salao', '#cabelo'],
		production_note: 'Selfie da equipe sorrindo no salao no fim do dia.',
	},
];

const routes: MockRoute[] = [
	{
		method: 'POST',
		pattern: /\/api\/businesses\/[^/]+\/posts:generateFromMessage$/,
		delay: 1500,
		response: MOCK_POST,
	},
	{
		method: 'POST',
		pattern: /\/api\/businesses\/[^/]+\/posts:generateIdeas$/,
		delay: 2000,
		response: MOCK_IDEAS,
	},
	{
		method: 'POST',
		pattern: /\/api\/businesses\/[^/]+\/posts:saveProactive$/,
		delay: 300,
		response: { ok: true },
	},
	{
		method: 'POST',
		pattern: /\/api\/messages:send$/,
		delay: 500,
		response: { ok: true },
	},
];

function sendJson(res: ServerResponse, data: unknown) {
	const body = JSON.stringify(data);
	res.writeHead(200, {
		'Content-Type': 'application/json',
		'Content-Length': Buffer.byteLength(body),
	});
	res.end(body);
}

export function mockApi(): Plugin {
	return {
		name: 'mock-api',
		configureServer(server) {
			// Runs before Vite's internal middleware (including the proxy)
			server.middlewares.use((req: IncomingMessage, res: ServerResponse, next: () => void) => {
				const url = req.url ?? '';
				const method = req.method ?? 'GET';

				for (const route of routes) {
					if (method === route.method && route.pattern.test(url)) {
						console.log(`[mock-api] ${method} ${url} -> ${route.delay}ms`);
						setTimeout(() => sendJson(res, route.response), route.delay);
						return;
					}
				}
				next();
			});
		},
	};
}
