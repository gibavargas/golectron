//go:build darwin && cgo

#import <AppKit/AppKit.h>

int electron_go_app_is_active(void) {
  @autoreleasepool {
    NSApplication *app = [NSApplication sharedApplication];
    return [app isActive] ? 1 : 0;
  }
}
