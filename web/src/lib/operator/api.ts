import { pb } from '$lib/pb';
import type {
	Business,
	GeneratedPost,
	Message,
	Post,
	ProfileSuggestion,
	ScheduledMessage,
	Service,
} from '$lib/types';

// --- Fetch ---

export async function fetchClients(): Promise<Business[]> {
	const res = await pb.collection('businesses').getList<Business>(1, 200, { sort: 'name' });
	return res.items;
}

export async function fetchMessages(): Promise<Message[]> {
	const res = await pb.collection('messages').getList<Message>(1, 500, { sort: 'created' });
	return res.items;
}

export async function fetchPosts(): Promise<Post[]> {
	const res = await pb.collection('posts').getList<Post>(1, 500, { sort: '-created' });
	return res.items;
}

export async function fetchScheduledMessages(): Promise<ScheduledMessage[]> {
	return (await pb.send('/api/scheduled-messages', { method: 'GET' })) as ScheduledMessage[];
}

export async function fetchSuggestionCounts(): Promise<Record<string, number>> {
	const res = await pb.collection('profile_suggestions').getList<ProfileSuggestion>(1, 500, {
		filter: 'dismissed = false',
		fields: 'business',
	});
	const counts: Record<string, number> = {};
	for (const s of res.items) {
		counts[s.business] = (counts[s.business] ?? 0) + 1;
	}
	return counts;
}

export async function fetchSuggestions(businessId: string): Promise<ProfileSuggestion[]> {
	const res = await pb.collection('profile_suggestions').getList<ProfileSuggestion>(1, 50, {
		filter: `business = "${businessId}" && dismissed = false`,
		sort: 'created',
	});
	return res.items;
}

// --- Mutations ---

export async function sendMessage(
	businessId: string,
	caption: string,
	hashtags: string[] = [],
	productionNote = '',
): Promise<void> {
	await pb.send('/api/messages:send', {
		method: 'POST',
		body: JSON.stringify({
			business_id: businessId,
			caption,
			hashtags: hashtags.join(' '),
			production_note: productionNote,
		}),
	});
}

export async function sendMediaMessage(
	businessId: string,
	file: File,
	caption: string,
): Promise<void> {
	const form = new FormData();
	form.append('business_id', businessId);
	form.append('file', file);
	form.append('caption', caption);
	await pb.send('/api/messages:sendMedia', { method: 'POST', body: form });
}

export async function describeMedia(file: File): Promise<string> {
	const form = new FormData();
	form.append('file', file);
	const res = (await pb.send('/api/media:describe', { method: 'POST', body: form })) as {
		description: string;
	};
	return res.description;
}

export async function generatePost(
	businessId: string,
	payload: Record<string, string>,
): Promise<GeneratedPost> {
	return (await pb.send(`/api/businesses/${businessId}/posts:generateFromMessage`, {
		method: 'POST',
		body: JSON.stringify(payload),
	})) as GeneratedPost;
}

export async function generateIdeas(businessId: string): Promise<GeneratedPost[]> {
	return (await pb.send(`/api/businesses/${businessId}/posts:generateIdeas`, {
		method: 'POST',
		body: JSON.stringify({}),
	})) as GeneratedPost[];
}

export async function saveProactivePost(
	businessId: string,
	caption: string,
	hashtags: string[],
	productionNote: string,
): Promise<void> {
	await pb.send(`/api/businesses/${businessId}/posts:saveProactive`, {
		method: 'POST',
		body: JSON.stringify({ caption, hashtags, production_note: productionNote }),
	});
}

export async function createBusiness(data: {
	user: string;
	name: string;
	type: string;
	city: string;
	state: string;
	phone: string;
	client_name: string;
	client_email: string;
	services: Service[];
	target_audience: string;
	brand_vibe: string;
	quirks: string;
}): Promise<Business> {
	return await pb.collection('businesses').create<Business>(data);
}

export async function updateBusiness(
	id: string,
	data: Partial<Omit<Business, 'id' | 'collectionId'>>,
): Promise<Business> {
	return await pb.collection('businesses').update<Business>(id, data);
}

export async function refreshBusiness(id: string): Promise<Business> {
	return await pb.collection('businesses').getOne<Business>(id);
}

export async function sendInvite(businessId: string): Promise<string> {
	const res = await pb.send(`/api/businesses/${businessId}/invites:send`, { method: 'POST' });
	return res.invite_url || '';
}

export async function cancelSubscription(businessId: string): Promise<void> {
	await pb.send(`/api/businesses/${businessId}/authorization:cancel`, { method: 'POST' });
}

export async function approveScheduledMessage(id: string): Promise<void> {
	await pb.send(`/api/scheduled-messages/${id}/approve`, { method: 'POST' });
}

export async function dismissScheduledMessage(id: string): Promise<void> {
	await pb.send(`/api/scheduled-messages/${id}/dismiss`, { method: 'POST' });
}

export async function acceptSuggestion(
	business: Business,
	sug: ProfileSuggestion,
): Promise<Record<string, unknown>> {
	const update: Record<string, unknown> = {};
	if (sug.field === 'services') {
		const parts = sug.suggestion.split('|');
		const name = parts[0]?.trim() ?? sug.suggestion;
		const price = parseFloat(parts[1] ?? '0') || 0;
		update.services = [...(business.services ?? []), { name, price_brl: price }];
	} else if (sug.field === 'quirks') {
		const existing = business.quirks ?? '';
		update.quirks = existing ? existing + '\n' + sug.suggestion : sug.suggestion;
	} else if (sug.field === 'target_audience') {
		const existing = business.target_audience ?? '';
		update.target_audience = existing ? existing + ', ' + sug.suggestion : sug.suggestion;
	} else if (sug.field === 'brand_vibe') {
		const existing = business.brand_vibe ?? '';
		update.brand_vibe = existing ? existing + ', ' + sug.suggestion : sug.suggestion;
	} else {
		return update;
	}

	await Promise.all([
		pb.collection('businesses').update<Business>(business.id, update),
		pb.collection('profile_suggestions').update(sug.id, { dismissed: true }),
	]);

	return update;
}

export async function dismissSuggestion(sugId: string): Promise<void> {
	await pb.collection('profile_suggestions').update(sugId, { dismissed: true });
}

export async function extractVoiceProfile(
	blob: Blob,
	businessType: string,
): Promise<{
	services?: { name: string; price_brl: number | null }[];
	target_audience?: string;
	brand_vibe?: string;
	quirks?: string[];
}> {
	const form = new FormData();
	form.append('audio', blob, 'recording.webm');
	form.append('business_type', businessType);
	const res = await fetch(`${pb.baseUrl}/api/businesses/profile:extract`, {
		method: 'POST',
		headers: { Authorization: pb.authStore.token },
		body: form,
	});
	if (!res.ok) throw new Error('extract failed');
	return await res.json();
}

// --- Subscriptions ---

export function subscribeMessages(
	onEvent: (action: string, record: Message) => void,
): Promise<() => void> {
	return pb.collection('messages').subscribe<Message>('*', (e) => {
		onEvent(e.action, e.record);
	});
}

export function subscribeBusinesses(
	onEvent: (action: string, record: Business) => void,
): Promise<() => void> {
	return pb.collection('businesses').subscribe<Business>('*', (e) => {
		onEvent(e.action, e.record);
	});
}

export function subscribePosts(
	onEvent: (action: string, record: Post) => void,
): Promise<() => void> {
	return pb.collection('posts').subscribe<Post>('*', (e) => {
		onEvent(e.action, e.record);
	});
}

export function subscribeScheduledMessages(onEvent: () => void): Promise<() => void> {
	return pb.collection('scheduled_messages').subscribe<ScheduledMessage>('*', () => {
		onEvent();
	});
}

export function subscribeSuggestions(
	onEvent: (action: string, record: ProfileSuggestion) => void,
): Promise<() => void> {
	return pb.collection('profile_suggestions').subscribe<ProfileSuggestion>('*', (e) => {
		onEvent(e.action, e.record);
	});
}
