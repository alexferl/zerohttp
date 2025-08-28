// Simple SPA router
const routes = {
  '/': {
    title: 'Home',
    content: `
            <h1>Welcome to the Home Page</h1>
            <p>This is a Single Page Application (SPA) test.</p>
            <div class="route-info">
                <strong>Current Route:</strong> / (home)<br>
                <strong>Note:</strong> This content is dynamically rendered by JavaScript.
            </div>
        `
  },
  '/about': {
    title: 'About',
    content: `
            <h1>About Us</h1>
            <p>This page demonstrates SPA routing where different URLs show different content without page reloads.</p>
            <div class="route-info">
                <strong>Current Route:</strong> /about<br>
                <strong>Behavior:</strong> The server serves index.html for this route, and JavaScript handles the content change.
            </div>
        `
  },
  '/contact': {
    title: 'Contact',
    content: `
            <h1>Contact Information</h1>
            <p>Get in touch with us!</p>
            <div class="route-info">
                <strong>Current Route:</strong> /contact<br>
                <strong>Testing:</strong> Try refreshing this page - you should stay on the contact page.
            </div>
        `
  }
};

function renderRoute(path) {
  const route = routes[path] || routes['/'];
  const contentEl = document.getElementById('content');
  const titleEl = document.querySelector('title');

  console.log('Rendering route:', path); // Debug logging

  // Update content
  contentEl.innerHTML = route.content;
  titleEl.textContent = `${route.title} - Test SPA`;

  // Update active nav link
  document.querySelectorAll('nav a').forEach(link => {
    link.classList.remove('active');
    const linkHref = link.getAttribute('href');
    if (linkHref === path || (path === '/' && linkHref === '/')) {
      link.classList.add('active');
    }
  });
}

function navigate(event, path) {
  event.preventDefault();
  console.log('Navigating to:', path); // Debug logging

  // Update browser URL without reload
  if (window.location.pathname !== path) {
    window.history.pushState({path: path}, '', path);
  }

  // Render the new content
  renderRoute(path);
}

// Handle browser back/forward buttons
window.addEventListener('popstate', (event) => {
  console.log('Popstate event:', event.state); // Debug logging
  const path = event.state?.path || window.location.pathname;
  renderRoute(path);
});

// Initial route render
document.addEventListener('DOMContentLoaded', () => {
  console.log('DOM loaded, initial path:', window.location.pathname); // Debug logging

  // Set up event listeners for navigation links
  document.querySelectorAll('nav a[href^="/"]').forEach(link => {
    const href = link.getAttribute('href');
    if (href && !href.startsWith('/api/')) {
      link.addEventListener('click', (event) => {
        navigate(event, href);
      });
    }
  });

  // Render initial route
  renderRoute(window.location.pathname);
});
