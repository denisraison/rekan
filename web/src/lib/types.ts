export interface User {
	id: string;
	name: string;
	email: string;
}

export interface Service {
	name: string;
	price_brl: number;
}

export type InviteStatus =
	| 'draft'
	| 'invited'
	| 'accepted'
	| 'active'
	| 'payment_failed'
	| 'cancelled';
export type Tier = 'basico' | 'parceiro' | 'profissional';
export type Commitment = 'mensal' | 'trimestral';

export interface Business {
	id: string;
	name: string;
	type: string;
	city: string;
	state: string;
	services: Service[];
	target_audience: string;
	brand_vibe: string;
	quirks: string;
	phone: string;
	onboarding_step: number;
	client_name: string;
	client_email: string;
	invite_token: string;
	invite_status: InviteStatus;
	invite_sent_at: string;
	authorization_id: string;
	customer_id: string;
	tier: Tier;
	commitment: Commitment;
	next_charge_date: string;
	charge_pending: boolean;
	terms_accepted_at: string;
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

export interface WAStatus {
	connected: boolean;
	qr?: string;
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
