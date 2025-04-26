/*
 * Copyright (c) 2006-2025 Pacman Development Team <pacman-dev@lists.archlinux.org>
 *
 * Converting the alpm function alpm_pkg_vercmp() to go
 * https://gitlab.archlinux.org/pacman/pacman/-/blob/master/lib/libalpm/version.c?ref_type=heads
 */

package alpm

import (
	"strings"
	"unicode"
)

// Some functions in this file have been adopted from the rpm source, notably
// 'rpmvercmp' located at lib/rpmvercmp.c and 'parseEVR' located at
// lib/rpmds.c. It was most recently updated against rpm version 4.8.1. Small
// modifications have been made to make it more consistent with the libalpm
// coding style.

// Split EVR into epoch, version, and release components.
// @param evr		[epoch:]version[-release] string
// @retval ep		pointer to epoch
// @retval vp		pointer to version
// @retval rp		pointer to release
func parseEVR(evr string) (epoch, version, release string) {
	parts := strings.SplitN(evr, ":", 2)
	if len(parts) == 2 {
		epoch = parts[0]
		versionRelease := parts[1]
		releaseParts := strings.SplitN(versionRelease, "-", 2)
		version = releaseParts[0]
		if len(releaseParts) == 2 {
			release = releaseParts[1]
		}
		if epoch == "" {
			epoch = "0"
		}
	} else {
		epoch = "0"
		releaseParts := strings.SplitN(parts[0], "-", 2)
		version = releaseParts[0]
		if len(releaseParts) == 2 {
			release = releaseParts[1]
		}
	}
	return
}

// Compare alpha and numeric segments of two versions.
// return 1: a is newer than b
//
//	 0: a and b are the same version
//	-1: b is newer than a
func rpmvercmp(a, b string) int {
	// easy comparison to see if versions are identical
	if a == b {
		return 0
	}

	one := []rune(a)
	two := []rune(b)
	ptr1 := 0
	ptr2 := 0
	ret := 0

	// loop through each version segment of str1 and str2 and compare them
	for ptr1 < len(one) && ptr2 < len(two) {
		start1 := ptr1
		start2 := ptr2

		for ptr1 < len(one) && !unicode.IsLetter(one[ptr1]) && !unicode.IsDigit(one[ptr1]) {
			ptr1++
		}
		for ptr2 < len(two) && !unicode.IsLetter(two[ptr2]) && !unicode.IsDigit(two[ptr2]) {
			ptr2++
		}

		// If we ran to the end of either, we are finished with the loop
		if ptr1 == len(one) || ptr2 == len(two) {
			break
		}

		// If the separator lengths were different, we are also finished
		if (ptr1 - start1) != (ptr2 - start2) {
			if (ptr1 - start1) < (ptr2 - start2) {
				ret = -1
			} else {
				ret = 1
			}
			goto cleanup
		}

		segStart1 := ptr1
		segStart2 := ptr2
		isNum := false

		// grab first completely alpha or completely numeric segment
		// leave ptr1 and ptr2 pointing to the start of the alpha or numeric
		// segment and walk to end of segment
		if unicode.IsDigit(one[ptr1]) {
			for ptr1 < len(one) && unicode.IsDigit(one[ptr1]) {
				ptr1++
			}
			for ptr2 < len(two) && unicode.IsDigit(two[ptr2]) {
				ptr2++
			}
			isNum = true
		} else {
			for ptr1 < len(one) && unicode.IsLetter(one[ptr1]) {
				ptr1++
			}
			for ptr2 < len(two) && unicode.IsLetter(two[ptr2]) {
				ptr2++
			}
			isNum = false
		}

		// this cannot happen, as we previously tested to make sure that
		// the first string has a non-null segment
		if segStart1 == ptr1 {
			ret = -1 // arbitrary
			goto cleanup
		}

		seg1 := string(one[segStart1:ptr1])
		seg2 := string(two[segStart2:ptr2])

		// take care of the case where the two version segments are
		// different types: one numeric, the other alpha (i.e. empty)
		// numeric segments are always newer than alpha segments
		// XXX See patch #60884 (and details) from bugzilla #50977.
		if segStart2 == ptr2 {
			if isNum {
				ret = 1
			} else {
				ret = -1
			}
			goto cleanup
		}

		if isNum {
			// this used to be done by converting the digit segments
			// to ints using atoi() - it's changed because long
			// digit segments can overflow an int - this should fix that.

			// throw away any leading zeros - it's a number, right?
			for len(seg1) > 1 && seg1[0] == '0' {
				seg1 = seg1[1:]
			}
			for len(seg2) > 1 && seg2[0] == '0' {
				seg2 = seg2[1:]
			}

			// whichever number has more digits wins
			if len(seg1) > len(seg2) {
				ret = 1
				goto cleanup
			}
			if len(seg2) > len(seg1) {
				ret = -1
				goto cleanup
			}
		}

		// strcmp will return which one is greater - even if the two
		// segments are alpha or if they are numeric.  don't return
		// if they are equal because there might be more segments to
		// compare
		if seg1 != seg2 {
			if seg1 < seg2 {
				ret = -1
			} else {
				ret = 1
			}
			goto cleanup
		}
	}

	// this catches the case where all numeric and alpha segments have
	// compared identically but the segment separating characters were
	// different
	if ptr1 == len(one) && ptr2 == len(two) {
		ret = 0
		goto cleanup
	}

	// the final showdown. we never want a remaining alpha string to
	// beat an empty string. the logic is a bit weird, but:
	// - if one is empty and two is not an alpha, two is newer.
	// - if one is an alpha, two is newer.
	// - otherwise one is newer.
	if ptr1 == len(one) && (ptr2 < len(two) && !unicode.IsLetter(two[ptr2])) ||
		(ptr1 < len(one) && unicode.IsLetter(one[ptr1])) {
		ret = -1
	} else {
		ret = 1
	}

cleanup:
	return ret
}

// SYMEXPORT alpm_pkg_vercmp(const char *a, const char *b)
func AlpmPkgVerCmp(a, b string) int {
	// ensure our strings are not nil
	if a == "" && b == "" {
		return 0
	} else if a == "" {
		return -1
	} else if b == "" {
		return 1
	}
	// another quick shortcut- if full version specs are equal
	if a == b {
		return 0
	}

	// Parse both versions into [epoch:]version[-release] triplets. We probably
	// don't need epoch and release to support all the same magic, but it is
	// easier to just run it all through the same code.
	epoch1, ver1, rel1 := parseEVR(a)
	epoch2, ver2, rel2 := parseEVR(b)

	ret := rpmvercmp(epoch1, epoch2)
	if ret == 0 {
		ret = rpmvercmp(ver1, ver2)
		if ret == 0 && rel1 != "" && rel2 != "" {
			ret = rpmvercmp(rel1, rel2)
		} else if ret == 0 && rel1 != "" && rel2 == "" {
			ret = 1
		} else if ret == 0 && rel1 == "" && rel2 != "" {
			ret = -1
		}
	}

	return ret
}
