import { expect, test } from '@playwright/test';
import { loginAsOperador, selectFirstClient, switchToGenerateMode } from './helpers';

test.use({ viewport: { width: 390, height: 844 } });

const overlay = '.absolute.inset-0.z-10';

async function generateIdeas(page: any) {
	await page.getByRole('button', { name: '3 ideias' }).click();
	await page.locator(overlay).getByText('Selecione ideias').waitFor({ timeout: 30000 });
}

test.describe('Multi-select ideas (Wave 4)', () => {
	test.beforeEach(async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);
		await switchToGenerateMode(page);
	});

	test('selecting multiple ideas shows send button with count', async ({ page }) => {
		await generateIdeas(page);
		const cards = page.locator(`${overlay} button.rounded-2xl`);
		await expect(cards.first()).toBeVisible();
		expect(await cards.count()).toBeGreaterThanOrEqual(2);

		await cards.nth(0).click();
		await cards.nth(1).click();
		await expect(page.getByRole('button', { name: 'Enviar 2 selecionadas' })).toBeVisible();
	});

	test('selecting one idea shows review button, opens overlay', async ({ page }) => {
		await generateIdeas(page);
		await page.locator(`${overlay} button.rounded-2xl`).first().click();

		const reviewBtn = page.getByRole('button', { name: 'Revisar e enviar' });
		await expect(reviewBtn).toBeVisible();
		await reviewBtn.click();
		await expect(page.getByText('Post gerado')).toBeVisible();

		// Voltar returns to idea list (scope to the review overlay which has "Post gerado")
		const reviewOverlay = page.locator(overlay, { hasText: 'Post gerado' });
		await reviewOverlay.locator('button', { hasText: 'Voltar' }).click();
		await expect(page.locator(overlay).getByText('Selecione ideias')).toBeVisible();
		const cards = page.locator(`${overlay} button.rounded-2xl`);
		expect(await cards.count()).toBeGreaterThanOrEqual(3);
	});

	test('Voltar from review returns to idea list', async ({ page }) => {
		await generateIdeas(page);
		const cards = page.locator(`${overlay} button.rounded-2xl`);
		await expect(cards.first()).toBeVisible();
		const count = await cards.count();

		await cards.first().click();
		await page.getByRole('button', { name: 'Revisar e enviar' }).click();
		await expect(page.getByText('Post gerado')).toBeVisible();

		const reviewOverlay = page.locator(overlay, { hasText: 'Post gerado' });
		await reviewOverlay.locator('button', { hasText: 'Voltar' }).click();
		await expect(page.locator(overlay).getByText('Selecione ideias')).toBeVisible();
		const cardsAfter = page.locator(`${overlay} button.rounded-2xl`);
		expect(await cardsAfter.count()).toBe(count);
	});

	test('review different idea after Voltar', async ({ page }) => {
		await generateIdeas(page);
		const cards = page.locator(`${overlay} button.rounded-2xl`);
		await expect(cards.first()).toBeVisible();

		// Select and review first idea
		await cards.nth(0).click();
		await page.getByRole('button', { name: 'Revisar e enviar' }).click();
		await expect(page.getByText('Post gerado')).toBeVisible();
		const firstCaption = await page.locator('textarea').inputValue();

		// Go back (scope to review overlay)
		const reviewOverlay = page.locator(overlay, { hasText: 'Post gerado' });
		await reviewOverlay.locator('button', { hasText: 'Voltar' }).click();
		await expect(page.locator(overlay).getByText('Selecione ideias')).toBeVisible();

		// Deselect first, select second
		await cards.nth(0).click();
		await cards.nth(1).click();
		await page.getByRole('button', { name: 'Revisar e enviar' }).click();
		await expect(page.getByText('Post gerado')).toBeVisible();
		const secondCaption = await page.locator('textarea').inputValue();

		expect(secondCaption.length).toBeGreaterThan(0);
	});

	test('deselecting an idea updates the count', async ({ page }) => {
		await generateIdeas(page);
		const cards = page.locator(`${overlay} button.rounded-2xl`);
		const count = await cards.count();
		for (let i = 0; i < count; i++) await cards.nth(i).click();
		await cards.nth(0).click();
		await expect(page.getByRole('button', { name: `Enviar ${count - 1} selecionadas` })).toBeVisible();
	});

	test('cancelar clears all selections', async ({ page }) => {
		await generateIdeas(page);
		await page.locator(`${overlay} button.rounded-2xl`).first().click();
		await page.getByRole('button', { name: 'Cancelar' }).click();
		await expect(page.getByRole('button', { name: 'Revisar e enviar' })).not.toBeVisible();
	});

	test('sending multiple ideas closes overlay', async ({ page }) => {
		await generateIdeas(page);
		const cards = page.locator(`${overlay} button.rounded-2xl`);
		await cards.nth(0).click();
		await cards.nth(1).click();
		await page.getByRole('button', { name: 'Enviar 2 selecionadas' }).click();
		await expect(page.locator(overlay).getByText('Selecione ideias')).not.toBeVisible({ timeout: 30000 });
	});
});
