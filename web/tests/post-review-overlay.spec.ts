import { expect, test } from '@playwright/test';
import { loginAsOperador, selectFirstClient, switchToGenerateMode } from './helpers';

const overlay = '.absolute.inset-0.z-10';

async function generatePost(page: any, prompt: string) {
	await page.locator('input[placeholder="Descreva o post..."]').fill(prompt);
	await page.getByRole('button', { name: 'Gerar' }).click();
	await page.getByText('Post gerado').waitFor({ timeout: 30000 });
}

test.describe('Post review overlay (Wave 3)', () => {
	test.beforeEach(async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);
		await switchToGenerateMode(page);
	});

	test('generate shows overlay with editable caption and actions', async ({ page }) => {
		await generatePost(page, 'Post sobre corte de cabelo');
		const textarea = page.locator('textarea');
		await expect(textarea).toBeVisible();
		expect((await textarea.inputValue()).length).toBeGreaterThan(0);
		await expect(page.getByText('Legenda')).toBeVisible();
		await expect(page.getByText('Hashtags')).toBeVisible();
		await expect(page.getByRole('button', { name: 'Enviar pelo WhatsApp' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Descartar' })).toBeVisible();
	});

	test('Voltar closes overlay, re-openable via chip', async ({ page }) => {
		await generatePost(page, 'Post sobre salao');
		await page.locator(`${overlay} button`, { hasText: 'Voltar' }).click();
		await expect(page.locator(overlay)).not.toBeVisible();
		const reopen = page.getByRole('button', { name: 'Ver post gerado' });
		await expect(reopen).toBeVisible();
		await reopen.click();
		await expect(page.locator(overlay)).toBeVisible();
	});

	test('Descartar clears everything', async ({ page }) => {
		await generatePost(page, 'Post sobre beleza');
		await page.getByRole('button', { name: 'Descartar' }).click();
		await expect(page.getByText('Post gerado')).not.toBeVisible();
		await expect(page.getByRole('button', { name: 'Ver post gerado' })).not.toBeVisible();
	});

	test('edited caption persists through Voltar round-trip', async ({ page }) => {
		await generatePost(page, 'Post sobre corte');
		const textarea = page.locator('textarea');
		await textarea.clear();
		await textarea.fill('Legenda editada');
		await page.locator(`${overlay} button`, { hasText: 'Voltar' }).click();
		await page.getByRole('button', { name: 'Ver post gerado' }).click();
		await expect(textarea).toHaveValue('Legenda editada');
	});

	test('Descartar from 3-ideas flow clears ideaDrafts', async ({ page }) => {
		// Use mobile viewport so the ideas overlay shows
		await page.setViewportSize({ width: 390, height: 844 });
		// Generate 3 ideas
		await page.getByRole('button', { name: '3 ideias' }).click();
		const ideasOverlay = page.locator(overlay).getByText('Selecione ideias');
		await ideasOverlay.waitFor({ timeout: 30000 });

		// Select one idea and review it
		const cards = page.locator(`${overlay} button.rounded-2xl`);
		await cards.first().click();
		await page.getByRole('button', { name: 'Revisar e enviar' }).click();
		await expect(page.getByText('Post gerado')).toBeVisible();

		// Discard
		await page.getByRole('button', { name: 'Descartar' }).click();

		// Idea list overlay should be gone
		await expect(ideasOverlay).not.toBeVisible();
		// Generate mode input should be back
		await expect(page.locator('input[placeholder="Descreva o post..."]')).toBeVisible();
	});
});
