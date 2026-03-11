import type { Business, Message } from "$lib/types";
import { pb } from "$lib/pb";

export function initials(business: Business): string {
  const name = business.client_name || business.name;
  return name.split(' ').slice(0, 2).map((w: string) => w[0]).join('').toUpperCase();
}

export function fmtTime(s: number): string {
  return `${Math.floor(s / 60)}:${(s % 60).toString().padStart(2, '0')}`;
}

export function profilePictureUrl(business: Business): string | null {
  if (!business.profile_picture) return null;
  return pb.files.getURL(
    { id: business.id, collectionId: business.collectionId },
    business.profile_picture,
  );
}

export function mediaUrl(msg: Message): string {
  return pb.files.getURL(
    { id: msg.id, collectionId: msg.collectionId },
    msg.media,
  );
}

export type MessageGroup = { date: Date; label: string; msgs: Message[] };

export function groupMessagesByDate(threadMessages: Message[]): MessageGroup[] {
  if (threadMessages.length === 0) return [];
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const yesterday = new Date(today.getTime() - 86400000);
  const groups: MessageGroup[] = [];
  let current: MessageGroup | null = null;
  for (const msg of threadMessages) {
    const d = new Date(msg.wa_timestamp || msg.created);
    d.setHours(0, 0, 0, 0);
    if (!current || current.date.getTime() !== d.getTime()) {
      let label: string;
      if (d.getTime() === today.getTime()) label = "Hoje";
      else if (d.getTime() === yesterday.getTime()) label = "Ontem";
      else
        label = d.toLocaleDateString("pt-BR", {
          weekday: "short",
          day: "numeric",
          month: "short",
        });
      current = { date: d, label, msgs: [] };
      groups.push(current);
    }
    current.msgs.push(msg);
  }
  return groups;
}
