// 安全工具函数

/**
 * HTML转义函数，防止XSS攻击
 * @param {string} str - 需要转义的字符串
 * @returns {string} - 转义后的安全字符串
 */
function escapeHtml(str) {
    if (typeof str !== 'string') {
        return '';
    }
    
    const htmlEscapes = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;',
        '/': '&#x2F;'
    };
    
    return str.replace(/[&<>"'\/]/g, function(match) {
        return htmlEscapes[match];
    });
}

/**
 * 安全地设置元素的HTML内容
 * @param {HTMLElement} element - 目标元素
 * @param {string} html - HTML内容
 */
function safeSetHTML(element, html) {
    if (!element) return;
    
    // 创建一个临时div来解析HTML
    const temp = document.createElement('div');
    temp.textContent = html;
    
    // 使用textContent而不是innerHTML
    element.textContent = temp.textContent;
}

/**
 * 创建安全的DOM元素
 * @param {string} tag - 标签名
 * @param {Object} attrs - 属性对象
 * @param {string} content - 文本内容
 * @returns {HTMLElement} - 创建的元素
 */
function createElement(tag, attrs = {}, content = '') {
    const element = document.createElement(tag);
    
    // 设置属性
    for (const [key, value] of Object.entries(attrs)) {
        if (key === 'className') {
            element.className = value;
        } else if (key.startsWith('data-')) {
            element.setAttribute(key, value);
        } else {
            element[key] = value;
        }
    }
    
    // 设置内容（安全地）
    if (content) {
        element.textContent = content;
    }
    
    return element;
}

/**
 * 安全地渲染列表
 * @param {Array} items - 数据项数组
 * @param {Function} renderItem - 渲染单个项的函数
 * @returns {DocumentFragment} - 文档片段
 */
function renderList(items, renderItem) {
    const fragment = document.createDocumentFragment();
    
    items.forEach((item, index) => {
        const element = renderItem(item, index);
        if (element instanceof Node) {
            fragment.appendChild(element);
        }
    });
    
    return fragment;
}

// 导出函数供其他文件使用
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        escapeHtml,
        safeSetHTML,
        createElement,
        renderList
    };
}