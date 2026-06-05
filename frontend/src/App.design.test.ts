/// <reference types="vitest/globals" />
import { readFileSync } from 'node:fs';

const appSource = readFileSync('src/App.tsx', 'utf8');
const appStyles = readFileSync('src/App.css', 'utf8');

function cssBlock(selector: string) {
    const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const match = appStyles.match(new RegExp(`${escaped}\\s*\\{([^}]*)\\}`, 'm'));
    expect(match, `${selector} CSS block`).not.toBeNull();
    return match?.[1] ?? '';
}

describe('Starsong design alignment contracts', () => {
    it('uses gold only for elevated confirmation actions', () => {
        expect(appSource.match(/className="primary-button gold-button"/g)).toHaveLength(2);
        expect(appSource).toContain('Confirm Anyway');
        expect(appSource).toContain('Go Live');
    });

    it('uses the Starsong accent color for selected segmented tabs', () => {
        const activeSegmentedButton = cssBlock('.segmented-control button.active');

        expect(activeSegmentedButton).toContain('border-color: var(--accent)');
        expect(activeSegmentedButton).toContain('var(--accent-soft)');
        expect(activeSegmentedButton).not.toContain('var(--accent-gold)');
    });

    it('keeps destination workspace tabs fixed-size and prevents Groups layout stretching', () => {
        expect(cssBlock('.destinations-workspace')).toContain('align-content: start');
        expect(cssBlock('.destinations-workspace')).toContain('grid-auto-rows: max-content');
        expect(cssBlock('.workspace-segmented-control')).toContain('grid-template-columns: repeat(2, minmax(0, 136px))');
        expect(cssBlock('.workspace-segmented-control button')).toContain('height: 40px');
    });

    it('keeps destination editor actions inside the settings panel flow', () => {
        const editorActions = cssBlock('.sticky-editor-actions');

        expect(editorActions).toContain('border-top: 1px solid var(--border-soft)');
        expect(editorActions).not.toContain('position: fixed');
        expect(editorActions).not.toContain('position: sticky');
    });

    it('keeps About identity content grouped without a redundant version badge', () => {
        expect(appSource).toContain('className="about-identity"');
        expect(appSource).toContain('className="about-product-row"');
        expect(appSource).toContain('className="about-version"');
        expect(appSource).not.toContain('Version {APP_VERSION}</span></div><div');
        expect(appSource).not.toContain('Recovery Center');
    });
});
