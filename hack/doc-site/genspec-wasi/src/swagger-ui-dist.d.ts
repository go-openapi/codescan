// swagger-ui-dist ships prebuilt bundles and no types.
//
// Declared here rather than pulled from DefinitelyTyped: the surface we use is one call with a
// handful of options, and the options that matter are the two that would otherwise reach the
// network - a types package would not make those safer, only better spelled.
declare module 'swagger-ui-dist/swagger-ui-es-bundle.js' {
  type Options = {
    domNode: HTMLElement;
    spec: unknown;
    /** null disables the badge, and with it the POST of the whole document to validator.swagger.io. */
    validatorUrl: null;
    /** Empty removes "Try it out", which fires real requests at whatever host the document names. */
    supportedSubmitMethods: string[];
    tryItOutEnabled?: boolean;
    deepLinking?: boolean;
    docExpansion?: 'list' | 'full' | 'none';
    defaultModelsExpandDepth?: number;
  };

  const SwaggerUIBundle: (options: Options) => unknown;
  export default SwaggerUIBundle;
}

declare module 'swagger-ui-dist/swagger-ui.css';
