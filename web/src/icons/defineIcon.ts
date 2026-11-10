import { h, type FunctionalComponent } from 'vue'

/**
 * Turns the text of a Material Symbols SVG file into a component. The file is a single path on a 0 -960 960 960 box; the
 * component draws it in the current text colour and takes the size from its class (`h-5 w-5`), like the icons it replaces.
 */
export function defineIcon(name: string, svg: string): FunctionalComponent {
  const viewBox = /viewBox="([^"]+)"/.exec(svg)?.[1]
  const path = /<path d="([^"]+)"/.exec(svg)?.[1]
  if (!viewBox || !path) throw new Error(`The icon ${name} is not a single-path SVG`)
  const Icon: FunctionalComponent = (_props, { attrs }) =>
    h('svg', { xmlns: 'http://www.w3.org/2000/svg', viewBox, fill: 'currentColor', focusable: 'false', 'data-icon': name, ...attrs }, [h('path', { d: path })])
  Icon.displayName = `Icon(${name})`
  return Icon
}
