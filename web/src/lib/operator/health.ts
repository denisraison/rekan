import type { Business, Message, Post } from '$lib/types';

export type ClientHealth = {
	daysSinceMsg: number;
	postsThisMonth: number;
	color: string;
};

export function computeClientHealth(
	clients: Business[],
	messages: Message[],
	posts: Post[],
): Record<string, ClientHealth> {
	const now = Date.now();
	const monthStart = new Date();
	monthStart.setDate(1);
	monthStart.setHours(0, 0, 0, 0);
	const monthStr = monthStart.toISOString();

	// Pre-group to avoid O(N*M) scans
	const lastIncoming = new Map<string, Message>();
	for (const m of messages) {
		if (m.direction !== 'incoming') continue;
		const prev = lastIncoming.get(m.business);
		if (!prev || m.created > prev.created) lastIncoming.set(m.business, m);
	}

	const postCounts = new Map<string, number>();
	for (const p of posts) {
		if (p.created >= monthStr) {
			postCounts.set(p.business, (postCounts.get(p.business) ?? 0) + 1);
		}
	}

	const health: Record<string, ClientHealth> = {};
	for (const client of clients) {
		const lastMsg = lastIncoming.get(client.id) ?? null;
		const daysSinceMsg = lastMsg
			? Math.floor((now - new Date(lastMsg.wa_timestamp || lastMsg.created).getTime()) / 86400000)
			: 999;

		const postsThisMonth = postCounts.get(client.id) ?? 0;

		let color = '#10B981'; // green
		if (daysSinceMsg >= 10)
			color = '#EF4444'; // red
		else if (daysSinceMsg >= 5) color = '#F59E0B'; // yellow

		health[client.id] = { daysSinceMsg, postsThisMonth, color };
	}
	return health;
}
