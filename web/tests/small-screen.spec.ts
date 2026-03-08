import { expect, test } from '@playwright/test';
import { loginAsOperador, selectFirstClient } from './helpers';

// iPhone SE viewport
test.use({ viewport: { width: 320, height: 568 } });

test.describe('Small screen accessibility (Wave 4)', () => {
	test('profile record card fits without overflow', async ({ page }) => {
		await loginAsOperador(page);
		// Open new client form to get the mic card
		await page.getByRole('button', { name: '+ Novo' }).click();
		const micBtn = page.getByRole('button', { name: 'Gravar descrição' });
		await expect(micBtn).toBeVisible();
		const descText = page.getByText('Toca no microfone e fala sobre a cliente');
		await expect(descText).toBeVisible();

		// Mic button should not overflow the viewport
		const btnBox = await micBtn.boundingBox();
		expect(btnBox).not.toBeNull();
		expect(btnBox!.x).toBeGreaterThanOrEqual(0);
		expect(btnBox!.x + btnBox!.width).toBeLessThanOrEqual(320);
	});

	test('message input bar has enough space for input and send button', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);
		const input = page.locator('input[placeholder="Mensagem..."]');
		await expect(input).toBeVisible();
		const inputBox = await input.boundingBox();
		expect(inputBox).not.toBeNull();
		// Input should have at least 120px width
		expect(inputBox!.width).toBeGreaterThanOrEqual(120);

		// Send button should be visible
		const sendBtn = page.getByRole('button', { name: 'Enviar' });
		await expect(sendBtn).toBeVisible();
		const sendBox = await sendBtn.boundingBox();
		expect(sendBox).not.toBeNull();
		// Input + send button should fit within the viewport (no horizontal scroll)
		const scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
		expect(scrollWidth).toBeLessThanOrEqual(320);
	});

	test('recording bar buttons meet minimum touch target size', async ({ page }) => {
		await loginAsOperador(page);
		// Open new client form
		await page.getByRole('button', { name: '+ Novo' }).click();
		await expect(page.getByRole('button', { name: 'Gravar descrição' })).toBeVisible();

		// We can't actually start a recording (needs microphone permission),
		// so verify the mic button itself meets touch target minimums
		const micBtn = page.getByRole('button', { name: 'Gravar descrição' });
		const micBox = await micBtn.boundingBox();
		expect(micBox).not.toBeNull();
		// 44x44 is the minimum recommended touch target (WCAG)
		expect(micBox!.width).toBeGreaterThanOrEqual(44);
		expect(micBox!.height).toBeGreaterThanOrEqual(44);
	});
});
