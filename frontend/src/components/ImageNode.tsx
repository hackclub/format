'use client'

import {
  $createParagraphNode,
  $createTextNode,
  $insertNodes,
  DecoratorNode,
  DOMConversionMap,
  DOMConversionOutput,
  DOMExportOutput,
  EditorConfig,
  LexicalNode,
  NodeKey,
  SerializedLexicalNode,
  Spread,
} from 'lexical'

export interface ImagePayload {
  altText: string
  src: string
  width?: number
  height?: number
}

export type SerializedImageNode = Spread<
  {
    altText: string
    src: string
    width?: number
    height?: number
  },
  SerializedLexicalNode
>

export class ImageNode extends DecoratorNode<JSX.Element> {
  __src: string
  __altText: string
  __width?: number
  __height?: number

  static getType(): string {
    return 'image'
  }

  static clone(node: ImageNode): ImageNode {
    return new ImageNode(
      node.__src,
      node.__altText,
      node.__width,
      node.__height,
      node.__key,
    )
  }

  constructor(
    src: string,
    altText: string,
    width?: number,
    height?: number,
    key?: NodeKey,
  ) {
    super(key)
    this.__src = src
    this.__altText = altText
    this.__width = width
    this.__height = height
  }

  exportDOM(): DOMExportOutput {
    const element = document.createElement('img')
    element.setAttribute('src', this.__src)
    element.setAttribute('alt', this.__altText)
    if (this.__width !== undefined) {
      element.setAttribute('width', this.__width.toString())
    }
    if (this.__height !== undefined) {
      element.setAttribute('height', this.__height.toString())
    }
    element.setAttribute('style', 'max-width: 100%; height: auto;')
    return { element }
  }

  static importDOM(): DOMConversionMap | null {
    return {
      img: () => ({
        conversion: convertImageElement,
        priority: 0,
      }),
    }
  }

  exportJSON(): SerializedImageNode {
    return {
      altText: this.getAltText(),
      src: this.getSrc(),
      width: this.__width,
      height: this.__height,
      type: 'image',
      version: 1,
    }
  }

  static importJSON(serializedNode: SerializedImageNode): ImageNode {
    const { altText, src, width, height } = serializedNode
    return $createImageNode({
      altText,
      src,
      width,
      height,
    })
  }

  getSrc(): string {
    return this.__src
  }

  getAltText(): string {
    return this.__altText
  }

  setAltText(altText: string): void {
    const writable = this.getWritable()
    writable.__altText = altText
  }

  createDOM(config: EditorConfig): HTMLElement {
    const img = document.createElement('img')
    img.src = this.__src
    img.alt = this.__altText
    img.style.maxWidth = '100%'
    img.style.height = 'auto'
    img.style.display = 'block'
    img.style.margin = '0.5rem 0'
    
    // Only set width/height if they are meaningful values (> 0)
    if (this.__width && this.__width > 0) {
      img.width = this.__width
    }
    if (this.__height && this.__height > 0) {
      img.height = this.__height
    }
    
    const { theme } = config
    if (theme.image) {
      img.className = theme.image
    }
    
    return img
  }

  updateDOM(): false {
    return false
  }

  decorate(): JSX.Element {
    const imgProps: React.ImgHTMLAttributes<HTMLImageElement> = {
      src: this.__src,
      alt: this.__altText,
      style: {
        maxWidth: '100%',
        height: 'auto',
        display: 'block',
        margin: '0.5rem 0'
      }
    }
    
    // Only set width/height if they are meaningful values
    if (this.__width && this.__width > 0) {
      imgProps.width = this.__width
    }
    if (this.__height && this.__height > 0) {
      imgProps.height = this.__height
    }
    
    return <img {...imgProps} />
  }
}

function convertImageElement(domNode: Node): null | DOMConversionOutput {
  if (domNode instanceof HTMLImageElement) {
    const { src, alt } = domNode
    
    // Only include dimensions if they are meaningful (> 0)
    const width = domNode.width > 0 ? domNode.width : undefined
    const height = domNode.height > 0 ? domNode.height : undefined
    
    const node = $createImageNode({ src, altText: alt || '', width, height })
    return { node }
  }
  return null
}

export function $createImageNode({ altText, src, width, height }: ImagePayload): ImageNode {
  return new ImageNode(src, altText, width, height)
}

export function $isImageNode(node: LexicalNode | null | undefined): node is ImageNode {
  return node instanceof ImageNode
}
