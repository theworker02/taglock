const root = document.documentElement;
const themeButton = document.querySelector('.theme-toggle');
const navButton = document.querySelector('.nav-toggle');
const navLinks = document.querySelector('#nav-links');

const storedTheme = localStorage.getItem('taglock-theme');
if (storedTheme === 'light' || storedTheme === 'dark') {
  root.dataset.theme = storedTheme;
} else if (window.matchMedia('(prefers-color-scheme: light)').matches) {
  root.dataset.theme = 'light';
}

themeButton?.addEventListener('click', () => {
  const next = root.dataset.theme === 'light' ? 'dark' : 'light';
  root.dataset.theme = next;
  localStorage.setItem('taglock-theme', next);
});

navButton?.addEventListener('click', () => {
  const open = navLinks.classList.toggle('open');
  navButton.setAttribute('aria-expanded', String(open));
});

navLinks?.querySelectorAll('a').forEach((link) => {
  link.addEventListener('click', () => {
    navLinks.classList.remove('open');
    navButton?.setAttribute('aria-expanded', 'false');
  });
});

document.querySelectorAll('[data-copy]').forEach((button) => {
  button.addEventListener('click', async () => {
    const target = document.getElementById(button.dataset.copy);
    if (!target) return;
    try {
      await navigator.clipboard.writeText(target.textContent.trim());
      const original = button.textContent;
      button.textContent = 'Copied';
      setTimeout(() => { button.textContent = original; }, 1400);
    } catch {
      const range = document.createRange();
      range.selectNodeContents(target);
      const selection = window.getSelection();
      selection.removeAllRanges();
      selection.addRange(range);
    }
  });
});
