export interface User {
	id: string;
	name: string;
	email: string;
	subscription_status: 'trial' | 'active' | 'past_due' | 'cancelled';
	subscription_id: string;
	generations_used: number;
}

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
	phone: string;
	onboarding_step: number;
}

export interface GeneratedPost {
	caption: string;
	hashtags: string[];
	production_note: string;
}

export interface Message {
	id: string;
	business: string;
	phone: string;
	type: 'text' | 'audio' | 'image';
	content: string;
	media: string;
	direction: 'incoming' | 'outgoing';
	wa_timestamp: string;
	wa_message_id: string;
	created: string;
	collectionId: string;
}

export interface Post {
	id: string;
	business: string;
	caption: string;
	hashtags: string[];
	production_note: string;
	role: string;
	hook: string;
	batch_id: string;
	edited: boolean;
	created: string;
}
