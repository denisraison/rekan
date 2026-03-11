import { expect, test, type Page, type Locator } from '@playwright/test';
import { loginAsOperador, selectFirstClient, switchToGenerateMode } from './helpers';

/**
 * Simulate Android "large text" accessibility setting.
 * Android scales the root font-size (affecting rem/em units).
 * Tailwind classes like text-sm, px-3, py-3 are all rem-based,
 * so they grow with the root font-size.
 */
async function setLargeText(page: Page) {
	// Android "Large" = ~1.3x, root goes from 16px to ~21px
	await page.addStyleTag({ content: 'html { font-size: 21px !important; }' });
}

async function setXLText(page: Page) {
	// Android "Largest" = ~1.5x, root goes from 16px to 24px
	await page.addStyleTag({ content: 'html { font-size: 24px !important; }' });
}

/** Assert element's right edge is within the viewport. */
async function expectOnscreen(page: Page, locator: Locator, label: string) {
	const box = await locator.boundingBox();
	expect(box, `${label} has no bounding box`).not.toBeNull();
	const vw = await page.evaluate(() => window.innerWidth);
	expect(box!.x + box!.width, `${label} right edge (${Math.round(box!.x + box!.width)}) > viewport (${vw})`).toBeLessThanOrEqual(vw + 2);
	expect(box!.x, `${label} starts offscreen left`).toBeGreaterThanOrEqual(-2);
}

/** Assert element's bottom edge is within the viewport. */
async function expectNotClippedBottom(page: Page, locator: Locator, label: string) {
	const box = await locator.boundingBox();
	expect(box, `${label} has no bounding box`).not.toBeNull();
	const vh = await page.evaluate(() => window.innerHeight);
	expect(box!.y + box!.height, `${label} bottom (${Math.round(box!.y + box!.height)}) > viewport height (${vh})`).toBeLessThanOrEqual(vh + 2);
}

/** Assert element meets WCAG minimum touch target (44x44). */
async function expectTouchTarget(locator: Locator, label: string) {
	const box = await locator.boundingBox();
	expect(box, `${label} has no bounding box`).not.toBeNull();
	expect(box!.width, `${label} width < 44`).toBeGreaterThanOrEqual(44);
	expect(box!.height, `${label} height < 44`).toBeGreaterThanOrEqual(44);
}

// ---------------------------------------------------------------------------
// iPhone SE (320x568) — smallest common viewport, normal text
// ---------------------------------------------------------------------------
test.describe('Small screen (320x568)', () => {
	test.use({ viewport: { width: 320, height: 568 } });

	test('chat: Enviar button onscreen', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);
		const btn = page.getByRole('button', { name: 'Enviar' });
		await expect(btn).toBeVisible();
		await expectOnscreen(page, btn, 'Enviar');
		await expectNotClippedBottom(page, btn, 'Enviar');
	});

	test('chat: Post toggle onscreen and tappable', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);
		const btn = page.getByRole('button', { name: 'Post' });
		await expect(btn).toBeVisible();
		await expectOnscreen(page, btn, 'Post toggle');
		await expectTouchTarget(btn, 'Post toggle');
	});

	test('chat: input has usable width', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);
		const input = page.locator('input[placeholder="Escreve aqui..."]');
		const box = await input.boundingBox();
		expect(box).not.toBeNull();
		expect(box!.width, 'input too narrow').toBeGreaterThanOrEqual(100);
	});

	test('generate: Gerar button onscreen', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);
		await switchToGenerateMode(page);
		const btn = page.getByRole('button', { name: /Gerar/ });
		await expect(btn).toBeVisible();
		await expectOnscreen(page, btn, 'Gerar');
		await expectNotClippedBottom(page, btn, 'Gerar');
	});

	test('chat: attach button onscreen', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);
		const btn = page.getByRole('button', { name: 'Anexar arquivo' });
		await expect(btn).toBeVisible();
		await expectOnscreen(page, btn, 'attach');
	});
});

// ---------------------------------------------------------------------------
// Small screen + Android "Large" text (21px root = 1.3x)
// ---------------------------------------------------------------------------
test.describe('Small screen + Large text (21px)', () => {
	test.use({ viewport: { width: 320, height: 568 } });

	test('chat: Enviar button onscreen', async ({ page }) => {
		await loginAsOperador(page);
		await setLargeText(page);
		await selectFirstClient(page);
		const btn = page.getByRole('button', { name: 'Enviar' });
		await expect(btn).toBeVisible();
		await expectOnscreen(page, btn, 'Enviar');
		await expectNotClippedBottom(page, btn, 'Enviar');
	});

	test('chat: Post toggle onscreen and tappable', async ({ page }) => {
		await loginAsOperador(page);
		await setLargeText(page);
		await selectFirstClient(page);
		const btn = page.getByRole('button', { name: 'Post' });
		await expect(btn).toBeVisible();
		await expectOnscreen(page, btn, 'Post toggle');
		await expectTouchTarget(btn, 'Post toggle');
	});

	test('chat: input has usable width', async ({ page }) => {
		await loginAsOperador(page);
		await setLargeText(page);
		await selectFirstClient(page);
		const input = page.locator('input[placeholder="Escreve aqui..."]');
		const box = await input.boundingBox();
		expect(box).not.toBeNull();
		expect(box!.width, 'input too narrow').toBeGreaterThanOrEqual(80);
	});

	test('generate: Gerar button onscreen', async ({ page }) => {
		await loginAsOperador(page);
		await setLargeText(page);
		await selectFirstClient(page);
		await switchToGenerateMode(page);
		const btn = page.getByRole('button', { name: /Gerar/ });
		await expect(btn).toBeVisible();
		await expectOnscreen(page, btn, 'Gerar');
		await expectNotClippedBottom(page, btn, 'Gerar');
	});

	test('chat: Voltar button visible', async ({ page }) => {
		await loginAsOperador(page);
		await setLargeText(page);
		await selectFirstClient(page);
		const btn = page.locator('button', { hasText: 'Voltar' }).first();
		await expect(btn).toBeVisible();
	});

	test('chat: input bar not clipped at bottom', async ({ page }) => {
		await loginAsOperador(page);
		await setLargeText(page);
		await selectFirstClient(page);
		const input = page.locator('input[placeholder="Escreve aqui..."]');
		await expectNotClippedBottom(page, input, 'message input');
	});
});

// ---------------------------------------------------------------------------
// Small screen + Android "Largest" text (24px root = 1.5x)
// ---------------------------------------------------------------------------
test.describe('Small screen + XL text (24px)', () => {
	test.use({ viewport: { width: 320, height: 568 } });

	test('chat: Enviar button onscreen', async ({ page }) => {
		await loginAsOperador(page);
		await setXLText(page);
		await selectFirstClient(page);
		const btn = page.getByRole('button', { name: 'Enviar' });
		await expect(btn).toBeVisible();
		await expectOnscreen(page, btn, 'Enviar');
		await expectNotClippedBottom(page, btn, 'Enviar');
	});

	test('generate: Gerar button onscreen', async ({ page }) => {
		await loginAsOperador(page);
		await setXLText(page);
		await selectFirstClient(page);
		await switchToGenerateMode(page);
		const btn = page.getByRole('button', { name: /Gerar/ });
		await expect(btn).toBeVisible();
		await expectOnscreen(page, btn, 'Gerar');
		await expectNotClippedBottom(page, btn, 'Gerar');
	});

	test('chat: input still usable width', async ({ page }) => {
		await loginAsOperador(page);
		await setXLText(page);
		await selectFirstClient(page);
		const input = page.locator('input[placeholder="Escreve aqui..."]');
		const box = await input.boundingBox();
		expect(box).not.toBeNull();
		expect(box!.width, 'input too narrow at XL').toBeGreaterThanOrEqual(60);
	});

	test('chat: Post toggle onscreen', async ({ page }) => {
		await loginAsOperador(page);
		await setXLText(page);
		await selectFirstClient(page);
		const btn = page.getByRole('button', { name: 'Post' });
		await expect(btn).toBeVisible();
		await expectOnscreen(page, btn, 'Post toggle');
	});
});

// ---------------------------------------------------------------------------
// Tiny budget phone (280x480) — common budget Android
// ---------------------------------------------------------------------------
test.describe('Tiny screen (280x480)', () => {
	test.use({ viewport: { width: 280, height: 480 } });

	test('chat: Enviar button onscreen', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);
		const btn = page.getByRole('button', { name: 'Enviar' });
		await expect(btn).toBeVisible();
		await expectOnscreen(page, btn, 'Enviar');
		await expectNotClippedBottom(page, btn, 'Enviar');
	});

	test('generate: Gerar button onscreen', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);
		await switchToGenerateMode(page);
		const btn = page.getByRole('button', { name: /Gerar/ });
		await expect(btn).toBeVisible();
		await expectOnscreen(page, btn, 'Gerar');
		await expectNotClippedBottom(page, btn, 'Gerar');
	});

	test('chat: input has usable width', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);
		const input = page.locator('input[placeholder="Escreve aqui..."]');
		const box = await input.boundingBox();
		expect(box).not.toBeNull();
		expect(box!.width, 'input too narrow on tiny screen').toBeGreaterThanOrEqual(80);
	});

	test('client list: + Novo button visible', async ({ page }) => {
		await loginAsOperador(page);
		const btn = page.getByRole('button', { name: '+ Novo' });
		await expect(btn).toBeVisible();
		await expectOnscreen(page, btn, '+ Novo');
	});
});

// ---------------------------------------------------------------------------
// New client form
// ---------------------------------------------------------------------------
test.describe('Large text: new client form', () => {
	test.use({ viewport: { width: 320, height: 568 } });

	test('mic card fits on screen', async ({ page }) => {
		await loginAsOperador(page);
		await setLargeText(page);
		await page.getByRole('button', { name: '+ Novo' }).click();
		const micBtn = page.getByRole('button', { name: 'Gravar descrição' });
		await expect(micBtn).toBeVisible();
		await expectOnscreen(page, micBtn, 'mic button');
	});
});
