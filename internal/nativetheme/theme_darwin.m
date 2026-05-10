//go:build darwin && cgo

#import <AppKit/AppKit.h>

int electron_go_should_differentiate_without_color(void) {
  @autoreleasepool {
    NSWorkspace *workspace = [NSWorkspace sharedWorkspace];
    SEL selector = NSSelectorFromString(@"accessibilityDisplayShouldDifferentiateWithoutColor");
    if (![workspace respondsToSelector:selector]) {
      return 0;
    }
    BOOL (*sendBool)(id, SEL) = (BOOL (*)(id, SEL))[workspace methodForSelector:selector];
    return sendBool(workspace, selector) ? 1 : 0;
  }
}
