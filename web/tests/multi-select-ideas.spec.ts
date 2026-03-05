import { expect, test } from '@playwright/test';

test.use({ ignoreHTTPSErrors: true, baseURL: 'https://localhost:5173', viewport: { width: 390, height: 844 } });

async function loginAsOperador(page: any) {
	await page.goto('/entrar');
	await page.getByLabel('Email').fill('operador@rekan.local');
	await page.getByLabel('Senha').fill('senha1234567');
	await page.getByRole('button', { name: 'Entrar' }).click();
	await page.waitForURL('**/operador**');
	await page.waitForTimeout(2000);
}

async function selectFirstClient(page: any) {
	const clientButton = page.locator('button.text-left.border-b').first();
	await clientButton.click();
	await page.waitForTimeout(1000);
}

async function switchToGenerateMode(page: any) {
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await page.waitForTimeout(300);
}

async function generateIdeas(page: any) {
	const ideasBtn = page.getByRole('button', { name: '3 ideias' });
	await ideasBtn.click();
	// Wait for ideas to load (the overlay title changes from "Gerando ideias..." to "Selecione ideias")
	// Both mobile and desktop versions exist; use the mobile overlay (md:hidden)
	await page.locator('.absolute.inset-0.z-10').getByText('Selecione ideias').waitFor({ timeout: 30000 });
}

test.describe('Multi-select ideas (Wave 4)', () => {
	test.beforeEach(async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);
		await switchToGenerateMode(page);
	});

	test('selecting multiple ideas shows send button with count', async ({ page }) => {
		await generateIdeas(page);

		// Ideas should be visible as selectable cards
		const ideaCards = page.locator('.absolute.inset-0.z-10 button.rounded-2xl');
		await expect(ideaCards.first()).toBeVisible();
		const count = await ideaCards.count();
		expect(count).toBeGreaterThanOrEqual(2);

		// Select first two ideas
		await ideaCards.nth(0).click();
		await ideaCards.nth(1).click();

		await page.screenshot({ path: '/tmp/ideas-multi-select.png', fullPage: true });

		// "Enviar 2 selecionadas" button should appear
		const sendBtn = page.getByRole('button', { name: 'Enviar 2 selecionadas' });
		await expect(sendBtn).toBeVisible();
	});

	test('selecting one idea shows review button, opens overlay', async ({ page }) => {
		await generateIdeas(page);

		const ideaCards = page.locator('.absolute.inset-0.z-10 button.rounded-2xl');
		await ideaCards.first().click();

		// Single select shows "Revisar e enviar"
		const reviewBtn = page.getByRole('button', { name: 'Revisar e enviar' });
		await expect(reviewBtn).toBeVisible();

		// Clicking it opens the review overlay
		await reviewBtn.click();
		await expect(page.getByText('Post gerado')).toBeVisible();

		await page.screenshot({ path: '/tmp/ideas-single-review.png', fullPage: true });
	});

	test('deselecting an idea updates the count', async ({ page }) => {
		await generateIdeas(page);

		const ideaCards = page.locator('.absolute.inset-0.z-10 button.rounded-2xl');

		// Select three
		const count = await ideaCards.count();
		for (let i = 0; i < count; i++) {
			await ideaCards.nth(i).click();
		}

		// Deselect one
		await ideaCards.nth(0).click();

		const sendBtn = page.getByRole('button', { name: `Enviar ${count - 1} selecionadas` });
		await expect(sendBtn).toBeVisible();
	});

	test('cancelar clears all selections', async ({ page }) => {
		await generateIdeas(page);

		const ideaCards = page.locator('.absolute.inset-0.z-10 button.rounded-2xl');
		await ideaCards.first().click();

		// Cancelar button should be visible
		const cancelBtn = page.getByRole('button', { name: 'Cancelar' });
		await expect(cancelBtn).toBeVisible();
		await cancelBtn.click();

		// Send/review buttons should disappear
		await expect(page.getByRole('button', { name: 'Revisar e enviar' })).not.toBeVisible();
	});

	test('sending multiple ideas sends them as separate messages', async ({ page }) => {
		await generateIdeas(page);

		const ideaCards = page.locator('.absolute.inset-0.z-10 button.rounded-2xl');
		await ideaCards.nth(0).click();
		await ideaCards.nth(1).click();

		const sendBtn = page.getByRole('button', { name: 'Enviar 2 selecionadas' });
		await sendBtn.click();

		// After sending, the ideas overlay should close
		await expect(page.locator('.absolute.inset-0.z-10').getByText('Selecione ideias')).not.toBeVisible({ timeout: 30000 });

		await page.screenshot({ path: '/tmp/ideas-sent.png', fullPage: true });
	});
});
