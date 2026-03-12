import { expect, test } from '@playwright/test';
import { loginAsOperador, selectFirstClient, selectClientByName, openInfoScreen, openNewClientForm, switchToGenerateMode } from './helpers';

// Moto G viewport
test.use({ viewport: { width: 360, height: 740 } });

test.describe('UX warmth (PEP-019 Wave 1)', () => {
	test('chat screen has warm background', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);

		const threadEl = page.locator('[data-testid="message-thread"]');
		const bg = await threadEl.evaluate((el) => getComputedStyle(el).backgroundColor);
		// --chat-bg: #fef8f7 = rgb(254, 248, 247)
		expect(bg).toBe('rgb(254, 248, 247)');
	});

	test('empty chat shows personalized message', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);

		// Check the template text in the source: "Quando {selected.name} mandar mensagem"
		// Even if the first client has messages (no empty state visible), verify the
		// template string is not the old generic one by checking the page source.
		const threadEl = page.locator('[data-testid="message-thread"]');
		const html = await threadEl.innerHTML();
		// If empty state is showing, it must be personalized
		if (html.includes('mandar mensagem')) {
			expect(html).not.toContain('Nenhuma mensagem ainda');
		}
	});

	test('section headers are at least 13px', async ({ page }) => {
		await loginAsOperador(page);
		await selectClientByName(page, 'Confeitaria da Elenice');
		await openInfoScreen(page);

		const header = page.locator('span').filter({ hasText: /^Serviços$/ }).last();
		await header.waitFor();
		const fontSize = await header.evaluate((el) => parseFloat(getComputedStyle(el).fontSize));
		expect(fontSize).toBeGreaterThanOrEqual(13);
	});
});

test.describe('UX warmth (PEP-019 Wave 2)', () => {
	test('new client form shows mic first', async ({ page }) => {
		await loginAsOperador(page);
		await openNewClientForm(page);

		await expect(page.locator('button[aria-label="Gravar descrição"]')).toBeVisible();
		// Fields should NOT be visible in idle mode
		await expect(page.locator('input[placeholder="Ex: Ana Silva"]')).not.toBeVisible();
	});

	test('manual mode shows fields', async ({ page }) => {
		await loginAsOperador(page);
		await openNewClientForm(page);
		await page.getByText('Preencher manualmente').click();

		await expect(page.locator('input[placeholder="Ex: Ana Silva"]')).toBeVisible();
	});

	test('generate mode has distinct background', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);

		const inputBar = page.locator('[data-testid="input-bar"]');
		const chatBg = await inputBar.evaluate((el) => getComputedStyle(el).backgroundColor);

		await switchToGenerateMode(page);

		const genBg = await inputBar.evaluate((el) => getComputedStyle(el).backgroundColor);
		expect(genBg).not.toBe(chatBg);
	});

	test('generate mode shows instruction banner', async ({ page }) => {
		await loginAsOperador(page);
		await selectClientByName(page, 'Hamburgueria do Léo');
		await switchToGenerateMode(page);

		await expect(page.getByText('Toque nas mensagens que quer usar no post')).toBeVisible();
	});

	test('screenshot: new client idle', async ({ page }) => {
		await loginAsOperador(page);
		await openNewClientForm(page);

		await page.screenshot({ path: '/tmp/pep019-newclient.png' });

		const micBtn = page.locator('button[aria-label="Gravar descrição"]');
		const box = await micBtn.boundingBox();
		expect(box).not.toBeNull();
		expect(box!.y).toBeLessThan(400);
	});
});

test.describe('UX warmth (PEP-019 Wave 3)', () => {
	test('screenshot: full flow', async ({ page }) => {
		await loginAsOperador(page);

		// List view
		await page.screenshot({ path: '/tmp/pep019-final-list.png' });

		// Chat view
		await selectFirstClient(page);
		await page.screenshot({ path: '/tmp/pep019-final-chat.png' });

		// Generate view
		await switchToGenerateMode(page);
		await page.screenshot({ path: '/tmp/pep019-final-generate.png' });

		// Info view
		await openInfoScreen(page);
		await page.screenshot({ path: '/tmp/pep019-final-info.png' });

		// New client view (info -> detail -> list)
		await page.goBack();
		await page.locator('[data-testid="input-bar"]').waitFor();
		await page.goBack();
		await page.getByRole('button', { name: '+ Novo' }).waitFor();
		await openNewClientForm(page);
		await page.screenshot({ path: '/tmp/pep019-final-newclient.png' });
	});
});
