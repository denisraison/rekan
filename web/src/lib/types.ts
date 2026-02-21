export interface Service {
	name: string;
	price_brl: number;
}

export interface Business {
	id: string;
	user: string;
	name: string;
	type: string;
	city: string;
	state: string;
	description: string;
	services: Service[];
	target_audience: string;
	brand_vibe: string;
	quirks: string;
	onboarding_step: number;
}
