//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

void ds_set_dock_badge(int count) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSApplication *app = [NSApplication sharedApplication];
        if (count <= 0) {
            [[app dockTile] setBadgeLabel:nil];
        } else {
            NSString *label = [NSString stringWithFormat:@"%d", count];
            [[app dockTile] setBadgeLabel:label];
        }
    });
}
*/
import "C"

func setDockBadge(count int) {
	C.ds_set_dock_badge(C.int(count))
}
