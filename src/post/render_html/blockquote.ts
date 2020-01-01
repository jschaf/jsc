import { RenderDispatcher } from '//post/render_html/dispatch';
import { hastElem } from '//post/render_html/hast_nodes';
import { checkNodeType } from '//post/render_html/md_nodes';
import { HastRenderer } from '//post/render_html/render';
import { renderChildren } from '//post/render_html/renders';
import * as unist from 'unist';
import * as md_nodes from '//post/render_html/md_nodes';
import vfile from 'vfile';

export class BlockquoteRenderer implements HastRenderer {
  private constructor(private readonly dispatcher: RenderDispatcher) {}

  static create(dispatcher: RenderDispatcher): BlockquoteRenderer {
    return new BlockquoteRenderer(dispatcher);
  }

  render(node: unist.Node, vf: vfile.VFile): Error | unist.Node {
    checkNodeType(node, 'blockquote', md_nodes.isBlockquote);
    const childRenders = renderChildren(node, vf, this.dispatcher);
    return hastElem('blockquote', childRenders);
  }
}
