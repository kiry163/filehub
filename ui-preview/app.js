const viewButtons = document.querySelectorAll('[data-view]');
const views = document.querySelectorAll('.view');
const navItems = document.querySelectorAll('.nav-item');
const toast = document.querySelector('.toast');
const copyButtons = document.querySelectorAll('[data-copy]');
const previewTabs = document.querySelectorAll('[data-preview]');
const previewPanes = document.querySelectorAll('[data-preview-pane]');
const drawer = document.querySelector('.drawer');
const drawerBackdrop = document.querySelector('.drawer-backdrop');
const drawerOpenButtons = document.querySelectorAll('[data-drawer-open]');
const drawerCloseButtons = document.querySelectorAll('[data-drawer-close]');
const taskFilterTabs = document.querySelectorAll('[data-task-filter]');
const fullscreen = document.querySelector('.fullscreen');
const fullscreenBody = document.querySelector('.fullscreen-body');
const fullscreenOpenButton = document.querySelector('[data-fullscreen]');
const fullscreenCloseButtons = document.querySelectorAll('[data-fullscreen-close]');

const setActiveView = (viewId) => {
  views.forEach((view) => {
    view.classList.toggle('active', view.id === `view-${viewId}`);
  });

  navItems.forEach((item) => {
    item.classList.toggle('active', item.dataset.view === viewId);
  });
};

viewButtons.forEach((button) => {
  button.addEventListener('click', () => {
    const target = button.dataset.view;
    if (target) {
      setActiveView(target);
      window.scrollTo({ top: 0, behavior: 'smooth' });
    }
  });
});

const showToast = (message) => {
  if (!toast) return;
  toast.textContent = message;
  toast.classList.add('show');
  setTimeout(() => toast.classList.remove('show'), 1600);
};

copyButtons.forEach((button) => {
  button.addEventListener('click', async () => {
    const text = button.dataset.copy;
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      showToast('已复制链接');
    } catch (error) {
      showToast('复制失败，请手动复制');
    }
  });
});

previewTabs.forEach((tab) => {
  tab.addEventListener('click', () => {
    const target = tab.dataset.preview;
    previewTabs.forEach((item) => {
      item.classList.toggle('active', item.dataset.preview === target);
    });
    previewPanes.forEach((pane) => {
      pane.classList.toggle('active', pane.dataset.previewPane === target);
    });
  });
});

const openDrawer = () => {
  if (!drawer || !drawerBackdrop) return;
  drawer.classList.add('open');
  drawerBackdrop.classList.add('open');
};

const closeDrawer = () => {
  if (!drawer || !drawerBackdrop) return;
  drawer.classList.remove('open');
  drawerBackdrop.classList.remove('open');
};

drawerOpenButtons.forEach((button) => {
  button.addEventListener('click', openDrawer);
});

drawerCloseButtons.forEach((button) => {
  button.addEventListener('click', closeDrawer);
});

if (drawerBackdrop) {
  drawerBackdrop.addEventListener('click', closeDrawer);
}

document.addEventListener('keydown', (event) => {
  if (event.key === 'Escape') {
    closeDrawer();
    closeFullscreen();
  }
});

taskFilterTabs.forEach((tab) => {
  tab.addEventListener('click', () => {
    const target = tab.dataset.taskFilter;
    taskFilterTabs.forEach((item) => {
      item.classList.toggle('active', item.dataset.taskFilter === target);
    });
  });
});

const openFullscreen = () => {
  if (!fullscreen || !fullscreenBody) return;
  fullscreenBody.innerHTML = '';
  const activePane = document.querySelector('.preview-pane.active');
  if (!activePane) return;
  fullscreenBody.appendChild(activePane.cloneNode(true));
  fullscreen.classList.add('open');
  fullscreen.setAttribute('aria-hidden', 'false');
};

const closeFullscreen = () => {
  if (!fullscreen || !fullscreenBody) return;
  fullscreen.classList.remove('open');
  fullscreen.setAttribute('aria-hidden', 'true');
  fullscreenBody.innerHTML = '';
};

if (fullscreenOpenButton) {
  fullscreenOpenButton.addEventListener('click', openFullscreen);
}

fullscreenCloseButtons.forEach((button) => {
  button.addEventListener('click', closeFullscreen);
});

if (fullscreen) {
  fullscreen.addEventListener('click', (event) => {
    if (event.target === fullscreen) {
      closeFullscreen();
    }
  });
}
