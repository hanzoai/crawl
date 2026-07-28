// Shared Markdown Preview Modal Component for Crawl Assistant
// Used by both SchemaBuilder and Click2CrawlBuilder

class MarkdownPreviewModal {
  constructor(options = {}) {
    this.modal = null;
    this.markdownOptions = {
      includeImages: true,
      preserveTables: true,
      keepCodeFormatting: true,
      simplifyLayout: false,
      preserveLinks: true,
      addSeparators: true,
      includeXPath: false,
      textOnly: false,
      ...options
    };
    this.onGenerateMarkdown = null;
    this.currentMarkdown = '';
  }

  show(generateMarkdownCallback) {
    this.onGenerateMarkdown = generateMarkdownCallback;
    
    if (!this.modal) {
      this.createModal();
    }
    
    // Generate initial markdown
    this.updateContent();
    this.modal.style.display = 'block';
  }

  hide() {
    if (this.modal) {
      this.modal.style.display = 'none';
    }
  }

  createModal() {
    this.modal = document.createElement('div');
    this.modal.className = 'crawl-c2c-preview';
    this.modal.innerHTML = `
      <div class="crawl-preview-header">
        <div class="crawl-toolbar-dots">
          <span class="crawl-dot crawl-dot-red"></span>
          <span class="crawl-dot crawl-dot-yellow"></span>
          <span class="crawl-dot crawl-dot-green"></span>
        </div>
        <span class="crawl-preview-title">Markdown Preview</span>
        <button class="crawl-preview-close">×</button>
      </div>
      <div class="crawl-preview-options">
        <label><input type="checkbox" name="textOnly"> 👁️ Visual Text Mode (As You See)</label>
        <label><input type="checkbox" name="includeImages" checked> Include Images</label>
        <label><input type="checkbox" name="preserveTables" checked> Preserve Tables</label>
        <label><input type="checkbox" name="preserveLinks" checked> Preserve Links</label>
        <label><input type="checkbox" name="keepCodeFormatting" checked> Keep Code Formatting</label>
        <label><input type="checkbox" name="simplifyLayout"> Simplify Layout</label>
        <label><input type="checkbox" name="addSeparators" checked> Add Separators</label>
        <label><input type="checkbox" name="includeXPath"> Include XPath Headers</label>
      </div>
      <div class="crawl-preview-content">
        <div class="crawl-preview-tabs">
          <button class="crawl-tab active" data-tab="preview">Preview</button>
          <button class="crawl-tab" data-tab="markdown">Markdown</button>
          <button class="crawl-wrap-toggle" title="Toggle word wrap">↔️ Wrap</button>
        </div>
        <div class="crawl-preview-pane active" data-pane="preview"></div>
        <div class="crawl-preview-pane" data-pane="markdown"></div>
      </div>
      <div class="crawl-preview-actions">
        <button class="crawl-download-btn">Download .md</button>
        <button class="crawl-copy-markdown-btn">Copy Markdown</button>
        <button class="crawl-cloud-btn" disabled>Send to Cloud (Coming Soon)</button>
      </div>
    `;
    
    document.body.appendChild(this.modal);
    
    // Make modal draggable
    if (window.CRAWL_Utils && window.CRAWL_Utils.makeDraggable) {
      window.CRAWL_Utils.makeDraggable(this.modal);
    }
    
    // Position preview modal
    this.modal.style.position = 'fixed';
    this.modal.style.top = '50%';
    this.modal.style.left = '50%';
    this.modal.style.transform = 'translate(-50%, -50%)';
    this.modal.style.zIndex = '999999';
    
    this.setupEventListeners();
  }

  setupEventListeners() {
    // Close button
    this.modal.querySelector('.crawl-preview-close').addEventListener('click', () => {
      this.hide();
    });
    
    // Tab switching
    this.modal.querySelectorAll('.crawl-tab').forEach(tab => {
      tab.addEventListener('click', (e) => {
        const tabName = e.target.dataset.tab;
        this.switchTab(tabName);
      });
    });
    
    // Wrap toggle
    const wrapToggle = this.modal.querySelector('.crawl-wrap-toggle');
    wrapToggle.addEventListener('click', () => {
      const panes = this.modal.querySelectorAll('.crawl-preview-pane');
      panes.forEach(pane => {
        pane.classList.toggle('wrap');
      });
      wrapToggle.classList.toggle('active');
    });
    
    // Options change
    this.modal.querySelectorAll('input[type="checkbox"]').forEach(checkbox => {
      checkbox.addEventListener('change', async (e) => {
        this.markdownOptions[e.target.name] = e.target.checked;
        
        // Handle text-only mode dependencies
        if (e.target.name === 'textOnly' && e.target.checked) {
          const preserveLinksCheckbox = this.modal.querySelector('input[name="preserveLinks"]');
          if (preserveLinksCheckbox) {
            preserveLinksCheckbox.checked = false;
            preserveLinksCheckbox.disabled = true;
            this.markdownOptions.preserveLinks = false;
          }
          
          const includeImagesCheckbox = this.modal.querySelector('input[name="includeImages"]');
          if (includeImagesCheckbox) {
            includeImagesCheckbox.disabled = true;
          }
        } else if (e.target.name === 'textOnly' && !e.target.checked) {
          // Re-enable options when text-only is disabled
          const preserveLinksCheckbox = this.modal.querySelector('input[name="preserveLinks"]');
          if (preserveLinksCheckbox) {
            preserveLinksCheckbox.disabled = false;
          }
          
          const includeImagesCheckbox = this.modal.querySelector('input[name="includeImages"]');
          if (includeImagesCheckbox) {
            includeImagesCheckbox.disabled = false;
          }
        }
        
        // Update markdown content
        await this.updateContent();
      });
    });
    
    // Action buttons
    this.modal.querySelector('.crawl-copy-markdown-btn').addEventListener('click', () => {
      this.copyToClipboard();
    });
    
    this.modal.querySelector('.crawl-download-btn').addEventListener('click', () => {
      this.downloadMarkdown();
    });
  }

  switchTab(tabName) {
    // Update active tab
    this.modal.querySelectorAll('.crawl-tab').forEach(tab => {
      tab.classList.toggle('active', tab.dataset.tab === tabName);
    });
    
    // Update active pane
    this.modal.querySelectorAll('.crawl-preview-pane').forEach(pane => {
      pane.classList.toggle('active', pane.dataset.pane === tabName);
    });
  }

  async updateContent() {
    if (!this.onGenerateMarkdown) return;
    
    try {
      // Generate markdown with current options
      this.currentMarkdown = await this.onGenerateMarkdown(this.markdownOptions);
      
      // Update markdown pane
      const markdownPane = this.modal.querySelector('[data-pane="markdown"]');
      markdownPane.innerHTML = `<pre><code>${this.escapeHtml(this.currentMarkdown)}</code></pre>`;
      
      // Update preview pane
      const previewPane = this.modal.querySelector('[data-pane="preview"]');
      
      // Use marked.js if available
      if (window.marked) {
        marked.setOptions({
          gfm: true,
          breaks: true,
          tables: true,
          headerIds: false,
          mangle: false
        });
        
        const html = marked.parse(this.currentMarkdown);
        previewPane.innerHTML = `<div class="crawl-markdown-preview">${html}</div>`;
      } else {
        // Fallback
        previewPane.innerHTML = `<div class="crawl-markdown-preview"><pre>${this.escapeHtml(this.currentMarkdown)}</pre></div>`;
      }
    } catch (error) {
      console.error('Error generating markdown:', error);
      this.showNotification('Error generating markdown', 'error');
    }
  }

  async copyToClipboard() {
    try {
      await navigator.clipboard.writeText(this.currentMarkdown);
      this.showNotification('Markdown copied to clipboard!');
    } catch (err) {
      console.error('Failed to copy:', err);
      this.showNotification('Failed to copy. Please try again.', 'error');
    }
  }

  async downloadMarkdown() {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, -5);
    const filename = `crawl-export-${timestamp}.md`;
    
    // Create blob and download
    const blob = new Blob([this.currentMarkdown], { type: 'text/markdown' });
    const url = URL.createObjectURL(blob);
    
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    
    this.showNotification(`Downloaded ${filename}`);
  }

  showNotification(message, type = 'success') {
    const notification = document.createElement('div');
    notification.className = `crawl-notification crawl-notification-${type}`;
    notification.textContent = message;
    
    document.body.appendChild(notification);
    
    // Animate in
    setTimeout(() => notification.classList.add('show'), 10);
    
    // Remove after 3 seconds
    setTimeout(() => {
      notification.classList.remove('show');
      setTimeout(() => notification.remove(), 300);
    }, 3000);
  }

  escapeHtml(unsafe) {
    return unsafe
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#039;");
  }

  // Get current options
  getOptions() {
    return { ...this.markdownOptions };
  }

  // Update options programmatically
  setOptions(options) {
    this.markdownOptions = { ...this.markdownOptions, ...options };
    
    // Update checkboxes to reflect new options
    Object.entries(options).forEach(([key, value]) => {
      const checkbox = this.modal?.querySelector(`input[name="${key}"]`);
      if (checkbox && typeof value === 'boolean') {
        checkbox.checked = value;
      }
    });
  }

  // Cleanup
  destroy() {
    if (this.modal) {
      this.modal.remove();
      this.modal = null;
    }
    this.onGenerateMarkdown = null;
  }
}

// Export for use in other scripts
if (typeof window !== 'undefined') {
  window.MarkdownPreviewModal = MarkdownPreviewModal;
}