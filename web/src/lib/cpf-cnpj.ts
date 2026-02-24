export function maskCpfCnpj(digits: string): string {
	const d = digits.replace(/\D/g, '');
	if (d.length <= 11) {
		return d
			.replace(/(\d{3})(\d)/, '$1.$2')
			.replace(/(\d{3})(\d)/, '$1.$2')
			.replace(/(\d{3})(\d{1,2})$/, '$1-$2');
	}
	return d
		.replace(/(\d{2})(\d)/, '$1.$2')
		.replace(/(\d{3})(\d)/, '$1.$2')
		.replace(/(\d{3})(\d)/, '$1/$2')
		.replace(/(\d{4})(\d{1,2})$/, '$1-$2');
}

export function validateCpfCnpj(value: string): boolean {
	const d = value.replace(/\D/g, '');
	if (d.length === 11) return validateCpf(d);
	if (d.length === 14) return validateCnpj(d);
	return false;
}

function allSame(digits: string): boolean {
	return digits.split('').every((c) => c === digits[0]);
}

function validateCpf(d: string): boolean {
	if (allSame(d)) return false;

	let sum = 0;
	for (let i = 0; i < 9; i++) sum += Number(d[i]) * (10 - i);
	let check = 11 - (sum % 11);
	if (check >= 10) check = 0;
	if (Number(d[9]) !== check) return false;

	sum = 0;
	for (let i = 0; i < 10; i++) sum += Number(d[i]) * (11 - i);
	check = 11 - (sum % 11);
	if (check >= 10) check = 0;
	return Number(d[10]) === check;
}

function validateCnpj(d: string): boolean {
	if (allSame(d)) return false;

	const w1 = [5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2];
	let sum = 0;
	for (let i = 0; i < 12; i++) sum += Number(d[i]) * w1[i];
	let check = sum % 11 < 2 ? 0 : 11 - (sum % 11);
	if (Number(d[12]) !== check) return false;

	const w2 = [6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2];
	sum = 0;
	for (let i = 0; i < 13; i++) sum += Number(d[i]) * w2[i];
	check = sum % 11 < 2 ? 0 : 11 - (sum % 11);
	return Number(d[13]) === check;
}
