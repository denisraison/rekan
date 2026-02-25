import { expect, type Page, test } from '@playwright/test';

const TOKEN = 'test-token-abc123';
const ACCEPT_URL = `**/api/invites/${TOKEN}/accept`;

const BASE_INVITE = {
	business_name: 'Padaria Dona Elza',
	client_name: 'Ana',
	invite_status: 'invited',
	tier: 'parceiro',
	commitment: 'mensal',
	price: 108.9,
	commitment_months: 1,
};

// Valid CPF: 529.982.247-25
const VALID_CPF = '52998224725';

// Matches GET /api/invites/{token} but not /api/invites/{token}/accept
function isInviteGet(url: URL) {
	return url.pathname === `/api/invites/${TOKEN}`;
}

function mockInvite(page: Page, data: Record<string, unknown>, status = 200) {
	return page.route(isInviteGet, (route) => {
		route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(data) });
	});
}

function mockInviteError(page: Page, status: number) {
	return page.route(isInviteGet, (route) => {
		route.fulfill({
			status,
			contentType: 'application/json',
			body: JSON.stringify({ message: 'error' }),
		});
	});
}

// --- Invite page: /convite/[token] ---

test.describe('invite page', () => {
	test('invited status shows greeting, form fields, and submit button', async ({ page }) => {
		await mockInvite(page, BASE_INVITE);
		await page.goto(`/convite/${TOKEN}`);

		await expect(page.getByText('Ana', { exact: false })).toBeVisible();
		await expect(page.getByText('Padaria Dona Elza')).toBeVisible();
		await expect(page.getByText('Parceiro')).toBeVisible();
		await expect(page.getByLabel('CPF ou CNPJ')).toBeVisible();
		await expect(page.getByLabel('Li e aceito os Termos de Uso')).toBeVisible();
		await expect(page.getByRole('button', { name: /Aceitar/ })).toBeVisible();
	});

	test('active status shows success message and WhatsApp link', async ({ page }) => {
		await mockInvite(page, { ...BASE_INVITE, invite_status: 'active' });
		await page.goto(`/convite/${TOKEN}`);

		await expect(page.getByText('Tudo certo')).toBeVisible();
		await expect(page.getByRole('link', { name: /WhatsApp/ })).toHaveAttribute('href', /wa\.me/);
	});

	test('expired invite (410) shows link expirado', async ({ page }) => {
		await mockInviteError(page, 410);
		await page.goto(`/convite/${TOKEN}`);

		await expect(page.getByText('Link expirado')).toBeVisible();
		await expect(page.getByRole('link', { name: /WhatsApp/ })).toHaveAttribute('href', /wa\.me/);
	});

	test('invalid token (404) shows link invalido', async ({ page }) => {
		await mockInviteError(page, 404);
		await page.goto(`/convite/${TOKEN}`);

		await expect(page.getByText('Link inválido')).toBeVisible();
	});

	test('invalid CPF shows validation error, valid CPF clears it', async ({ page }) => {
		await mockInvite(page, BASE_INVITE);
		await page.goto(`/convite/${TOKEN}`);

		const input = page.getByLabel('CPF ou CNPJ');
		const submit = page.getByRole('button', { name: /Aceitar/ });

		await page.getByLabel('Li e aceito os Termos de Uso').check();

		await input.fill('12345678900');
		await submit.click();
		await expect(page.getByText('CPF ou CNPJ inválido')).toBeVisible();

		await input.fill('');
		await input.pressSequentially(VALID_CPF);
		await expect(page.getByText('CPF ou CNPJ inválido')).not.toBeVisible();
	});

	test('submitting without accepting terms shows error', async ({ page }) => {
		await mockInvite(page, BASE_INVITE);
		await page.goto(`/convite/${TOKEN}`);

		const input = page.getByLabel('CPF ou CNPJ');
		await input.fill('');
		await input.pressSequentially(VALID_CPF);
		await page.getByRole('button', { name: /Aceitar/ }).click();

		await expect(page.getByText('Você precisa aceitar os Termos de Uso.')).toBeVisible();
	});

	test('accept flow shows QR code for payment', async ({ page }) => {
		await mockInvite(page, BASE_INVITE);
		await page.route(ACCEPT_URL, (route) => {
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ qr_payload: '00020126580014br.gov.bcb.pix0136test-payload' }),
			});
		});

		await page.goto(`/convite/${TOKEN}`);

		await page.getByLabel('CPF ou CNPJ').pressSequentially(VALID_CPF);
		await page.getByLabel('Li e aceito os Termos de Uso').check();

		const [request] = await Promise.all([
			page.waitForRequest((req) => req.url().includes('/accept') && req.method() === 'POST'),
			page.getByRole('button', { name: /Aceitar/ }).click(),
		]);

		const body = JSON.parse(request.postData() ?? '{}');
		expect(body.cpf_cnpj).toBe(VALID_CPF);

		// QR code and copy button should appear
		await expect(page.getByText('Escaneie o QR Code')).toBeVisible();
		await expect(page.getByRole('button', { name: 'Copiar' })).toBeVisible();
		await expect(page.getByText('Aguardando confirmação do pagamento')).toBeVisible();
	});

	test('accepted status with qr_payload shows QR code', async ({ page }) => {
		await mockInvite(page, {
			...BASE_INVITE,
			invite_status: 'accepted',
			qr_payload: '00020126580014br.gov.bcb.pix0136test-payload',
		});
		await page.goto(`/convite/${TOKEN}`);

		await expect(page.getByText('Escaneie o QR Code')).toBeVisible();
		await expect(page.locator('.animate-spin')).toBeVisible();
	});

	test('polling transitions from accepted to active', async ({ page }) => {
		let callCount = 0;
		await page.route(isInviteGet, (route) => {
			callCount++;
			if (callCount <= 1) {
				route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						...BASE_INVITE,
						invite_status: 'accepted',
						qr_payload: '00020126580014br.gov.bcb.pix0136test-payload',
					}),
				});
			} else {
				route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ ...BASE_INVITE, invite_status: 'active', client_name: 'Ana' }),
				});
			}
		});

		await page.goto(`/convite/${TOKEN}`);

		await expect(page.getByText('Aguardando confirmação do pagamento')).toBeVisible();
		await expect(page.getByText('Tudo certo')).toBeVisible({ timeout: 10000 });
	});
});

// --- Confirmation page redirects to invite page ---

test.describe('confirmation page', () => {
	test('redirects to invite page', async ({ page }) => {
		await mockInvite(page, { ...BASE_INVITE, invite_status: 'active' });
		await page.goto(`/convite/${TOKEN}/confirmacao`);

		await page.waitForURL(`**/convite/${TOKEN}`);
		await expect(page.getByText('Tudo certo')).toBeVisible();
	});
});
