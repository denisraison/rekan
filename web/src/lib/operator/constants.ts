export const BUSINESS_TYPES = [
	'Salão de Beleza',
	'Restaurante',
	'Personal Trainer',
	'Nail Designer',
	'Confeitaria',
	'Barbearia',
	'Loja de Roupas',
	'Pet Shop',
	'Banda Musical',
	'Estúdio de Tatuagem',
	'Hamburgueria',
	'Loja de Açaí',
	'Outro',
] as const;

export const STATES = [
	'AC',
	'AL',
	'AP',
	'AM',
	'BA',
	'CE',
	'DF',
	'ES',
	'GO',
	'MA',
	'MT',
	'MS',
	'MG',
	'PA',
	'PB',
	'PR',
	'PE',
	'PI',
	'RJ',
	'RN',
	'RS',
	'RO',
	'RR',
	'SC',
	'SP',
	'SE',
	'TO',
] as const;

export type NudgeTemplate = {
	minDays: number;
	maxDays: number;
	template: string;
};

export const NUDGE_TEMPLATES: NudgeTemplate[] = [
	{
		minDays: 5,
		maxDays: 7,
		template: 'Oi {name}, como foi a semana? Tem algo legal pra gente postar?',
	},
	{
		minDays: 8,
		maxDays: 14,
		template: '{name}, tudo bem? Faz um tempinho que a gente não posta. Bora preparar algo novo?',
	},
	{
		minDays: 15,
		maxDays: Infinity,
		template:
			'{name}, vi que faz um tempo! Quer retomar? Posso te mandar ideias de conteúdo pra essa semana.',
	},
];

export type SeasonalDate = {
	month: number;
	day: number;
	label: string;
	niches: string[];
	template: string;
};

// Moveable holidays (Carnaval, Páscoa, Dia das Mães) hardcoded for 2026
export const SEASONAL_DATES: SeasonalDate[] = [
	{
		month: 2,
		day: 14,
		label: 'Carnaval',
		niches: ['Salão de Beleza', 'Barbearia', 'Personal Trainer', 'Nail Designer'],
		template: '{name}, Carnaval tá chegando! Vamos preparar posts especiais?',
	},
	{
		month: 3,
		day: 8,
		label: 'Dia da Mulher',
		niches: ['Salão de Beleza', 'Nail Designer', 'Confeitaria', 'Loja de Roupas'],
		template: '{name}, Dia da Mulher vem ai! Que tal um post com promo especial?',
	},
	{
		month: 4,
		day: 5,
		label: 'Páscoa',
		niches: ['Confeitaria', 'Restaurante', 'Hamburgueria', 'Loja de Açaí'],
		template: '{name}, Páscoa tá chegando! Vamos montar os posts das encomendas?',
	},
	{
		month: 5,
		day: 10,
		label: 'Dia das Mães',
		niches: ['Salão de Beleza', 'Confeitaria', 'Nail Designer', 'Loja de Roupas', 'Restaurante'],
		template: '{name}, Dia das Mães daqui a pouco! Bora preparar posts de presente e promo?',
	},
	{
		month: 6,
		day: 12,
		label: 'Dia dos Namorados',
		niches: ['Confeitaria', 'Restaurante', 'Hamburgueria', 'Salão de Beleza', 'Loja de Roupas'],
		template: '{name}, Dia dos Namorados vem ai! Vamos criar posts romanticos pro seu negocio?',
	},
	{
		month: 6,
		day: 13,
		label: 'Festas Juninas',
		niches: ['Confeitaria', 'Restaurante', 'Hamburgueria', 'Banda Musical'],
		template: '{name}, Junho ta ai! Vamos postar algo com tema junino?',
	},
	{
		month: 9,
		day: 1,
		label: 'Dia do Educador Físico',
		niches: ['Personal Trainer'],
		template: '{name}, vem ai o Dia do Educador Fisico! Bora fazer um post especial?',
	},
	{
		month: 10,
		day: 1,
		label: 'Início do Verão',
		niches: ['Personal Trainer', 'Loja de Açaí'],
		template: '{name}, verao chegando! Momento perfeito pra postar sobre preparacao e resultados.',
	},
	{
		month: 10,
		day: 12,
		label: 'Dia das Crianças',
		niches: ['Confeitaria', 'Pet Shop', 'Loja de Roupas'],
		template: '{name}, Dia das Criancas ta perto! Vamos criar posts com ofertas kids?',
	},
	{
		month: 12,
		day: 19,
		label: 'Dia do Cabeleireiro',
		niches: ['Salão de Beleza', 'Barbearia'],
		template:
			'{name}, Dia do Cabeleireiro chegando! Que tal um post especial celebrando a profissao?',
	},
	{
		month: 12,
		day: 25,
		label: 'Natal',
		niches: [],
		template:
			'{name}, Natal chegando! Vamos preparar posts com ofertas e mensagem de final de ano?',
	},
	{
		month: 12,
		day: 31,
		label: 'Réveillon',
		niches: ['Salão de Beleza', 'Barbearia', 'Nail Designer', 'Personal Trainer', 'Loja de Roupas'],
		template: '{name}, Réveillon vem aí! Bora postar sobre agendamento e preparação?',
	},
];

const SEASONAL_DATES_SORTED = [...SEASONAL_DATES].sort((a, b) =>
	a.month !== b.month ? a.month - b.month : a.day - b.day,
);

export type UpcomingDate = SeasonalDate & { daysUntil: number };

export type NearestSeasonal = SeasonalDate & { daysUntil: number; eligibleCount: number };

/**
 * Find the nudge template tier for a given days-since-message count.
 * Returns null if the client is active (< 5 days) or has no messages (999).
 */
export function findNudgeTier(daysSinceMsg: number): NudgeTemplate | null {
	if (daysSinceMsg < 5 || daysSinceMsg === 999) return null;
	return (
		NUDGE_TEMPLATES.find((t) => daysSinceMsg >= t.minDays && daysSinceMsg <= t.maxDays) ??
		NUDGE_TEMPLATES[NUDGE_TEMPLATES.length - 1]
	);
}

/** Resolve a nudge/seasonal template with the client's first name. */
export function resolveTemplate(
	template: string,
	clientName: string,
	businessName: string,
): string {
	const firstName = clientName ? clientName.split(' ')[0] : businessName;
	return template.replace('{name}', firstName);
}

/** Resolve a seasonal date to its next occurrence, returning null if outside the window. */
function resolveDate(sd: SeasonalDate, now: Date, limit: Date): Date | null {
	const year = now.getFullYear();
	const date = new Date(year, sd.month - 1, sd.day);
	if (date < now) date.setFullYear(year + 1);
	return date <= limit ? date : null;
}

/**
 * Find the nearest seasonal date relevant to any of the given business types,
 * within the next 30 days.
 */
export function findNearestSeasonal(businessTypes: string[]): NearestSeasonal | null {
	const now = new Date();
	const limit = new Date(now.getTime() + 30 * 86400000);
	for (const sd of SEASONAL_DATES_SORTED) {
		const date = resolveDate(sd, now, limit);
		if (!date) continue;
		const eligibleCount = businessTypes.filter(
			(t) => sd.niches.length === 0 || sd.niches.includes(t),
		).length;
		if (eligibleCount > 0) {
			return {
				...sd,
				daysUntil: Math.ceil((date.getTime() - now.getTime()) / 86400000),
				eligibleCount,
			};
		}
	}
	return null;
}

/**
 * Get upcoming seasonal dates relevant to a specific business type,
 * within the next 30 days.
 */
export function getUpcomingDates(businessType: string): UpcomingDate[] {
	const now = new Date();
	const limit = new Date(now.getTime() + 30 * 86400000);

	return SEASONAL_DATES.flatMap((d) => {
		if (d.niches.length > 0 && !d.niches.includes(businessType)) return [];
		const date = resolveDate(d, now, limit);
		if (!date) return [];
		const daysUntil = Math.ceil((date.getTime() - now.getTime()) / 86400000);
		return [{ ...d, daysUntil }];
	}).sort((a, b) => a.daysUntil - b.daysUntil);
}

// Aliases for component-friendly imports
export { BUSINESS_TYPES as businessTypes, STATES as states };
